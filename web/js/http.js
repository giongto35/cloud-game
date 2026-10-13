/**
 * JSON HTTP client for the coordinator API.
 * Errors are thrown as {status, code, message} with pt-BR messages.
 */
export class ApiError extends Error {
    constructor(status, code, message) {
        super(message);
        this.status = status;
        this.code = code;
    }
}

const request = async (method, url, body) => {
    let res;
    try {
        res = await fetch(url, {
            method,
            credentials: 'same-origin',
            headers: body !== undefined ? {'Content-Type': 'application/json'} : {},
            body: body !== undefined ? JSON.stringify(body) : undefined,
        });
    } catch {
        throw new ApiError(0, 'network', 'Sem conexão com o servidor. Verifique sua internet.');
    }
    let data = null;
    try {
        data = await res.json();
    } catch {
        // empty or non-JSON body
    }
    if (!res.ok) {
        const e = data?.error || {};
        throw new ApiError(res.status, e.code || 'error', e.message || 'Algo deu errado. Tente novamente.');
    }
    return data;
};

export const http = {
    get: (url) => request('GET', url),
    post: (url, body = {}) => request('POST', url, body),
    put: (url, body = {}) => request('PUT', url, body),
    del: (url) => request('DELETE', url, {}),
};
