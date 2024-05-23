import { blake2b } from 'blakejs';
import { Buffer } from 'buffer';

export enum Flag {
    UNKNOWN,
    INVISIBLE,
    VISIBLE,
    READONLY,
    READWRITE
}

export interface Effect {
    definition: string,
    direct: boolean,
    cause: string
}

export interface DirItem {
    name: string,
    time: Date,
    flag: Flag,
    effect: Effect | null
}

export interface FileItem extends DirItem {
    size: number,
    assoc: string
}

export type Result<T> = {
    ok: true,
    data: T
} | {
    ok: false,
    error?: string
}

export type TreeResult = Result<{ 
    files: FileItem[], 
    dirs: DirItem[] 
}>

export type Progress = (index: number, total: number) => void;

export interface Backend {
    tree(...path: string[]): Promise<TreeResult>
    iconSrc(...path: string[]): string
    dirIconSrc: string
    fileUrl(...path: string[]): string
    downloadUrl(...path: string[]): string
    upload(blob: Blob, callback?: Progress, ...path: string[]): Promise<void>
    copy(from: string[], to: string[]): Promise<void>
    move(from: string[], to: string[]): Promise<void>
    delete(...path: string[]): Promise<void>
}

const base = import.meta.env.DEV ? "http://localhost:8080" : "";

const CHUNK_SIZE = 1024*1024;

function getWsBase(): string {
    if (import.meta.env.DEV) {
        return "ws://localhost:8080";
    }
    return `ws://${location.host}`
}

const backend: Backend = {
    async tree(...path) {
        const fullPath = path.join('/');
        const resp = await fetch(`${base}/tree/${fullPath}`);
        if (resp.status !== 200) {
            throw resp.status;
        }
        const data = await resp.json();
        data.data.files = data.data.files.map((f: any) => ({ ...f, time: new Date(f.time)}));
        data.data.dirs = data.data.dirs.map((f: any) => ({ ...f, time: new Date(f.time)}));
        return data;
    },
    iconSrc(...path) {
        const fullPath = path.join('/');
        return `${base}/fileicon/${fullPath}`
    },
    dirIconSrc: `${base}/foldericon`,
    fileUrl(...path) {
        const fullPath = path.join('/');
        return `${base}/file/${fullPath}`;
    },
    downloadUrl(...path) {
        return this.fileUrl(...path) + '?download=true';
    },
    upload(blob, callback, ...path) {
        const fullPath = path.join('/');
        const count = Math.ceil(blob.size / CHUNK_SIZE);
        const key = new Buffer(32);
        crypto.getRandomValues(key);
        const ws = new WebSocket(`${getWsBase()}/upload/${fullPath}`);
        ws.addEventListener('open', () => {
            ws.send(JSON.stringify({
                key: key.toString('hex'),
                count
            }))
        });
        return new Promise((resolve, reject) => {
            let index = -1;
            ws.addEventListener('message', ev => {
                const ok = JSON.parse(ev.data);
                if (ok) {
                    callback?.(index, count);
                    if (++index >= count) {
                        return;
                    }
                }
                blob
                    .slice(index*CHUNK_SIZE, (index+1)*CHUNK_SIZE)
                    .arrayBuffer()
                    .then(arrayBuffer => {
                        const data = Buffer.from(arrayBuffer);
                        const mac = blake2b(data, key, 32);
                        ws.send(Buffer.concat([mac, data]))
                    })
            });
            ws.addEventListener('error', () => {
                reject();
            });
            ws.addEventListener('close', ev => {
                if (ev.code != 1000) {
                    reject(ev.code);
                }
                else {
                    resolve();
                }
            });
        })
    },
    async copy(from, to) {
        const resp = await fetch(`${base}/copy`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                from, to
            })
        });
        if (resp.status !== 200) {
            throw resp.status;
        }
    },
    async move(from, to) {
        const resp = await fetch(`${base}/move`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                from, to
            })
        });
        if (resp.status !== 200) {
            throw resp.status;
        }
    },
    async delete(...path) {
        const fullPath = path.join('/');
        const resp = await fetch(`${base}/delete/${fullPath}`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            }
        });
        if (resp.status !== 200) {
            throw resp.status;
        }
    },
}

export default backend;