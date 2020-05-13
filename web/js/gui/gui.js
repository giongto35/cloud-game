const gui = (() => {

    const _option = (text = '', selected = false) => {
        const el = document.createElement('option');
        el.innerText = text;
        if (selected) el.selected = true;

        return el;
    }

    const select = (callback = () => {console.log('yolo')}, values = [], current = '') => {
        const el = document.createElement('div');

        const select = document.createElement('select');
        select.onchange = callback;
        el.append(select);

        select.append(_option('Make your choice', current === ''))
        values.forEach(v => {
            select.append(_option(v, current === v));
        });

        return el;
    }

    return {
        select,
    }
})(document);
