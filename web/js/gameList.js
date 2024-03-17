import {
    sub,
    MENU_PRESSED,
    MENU_RELEASED
} from 'event';
import {gui} from 'gui';

const TOP_POSITION = 102
const SELECT_THRESHOLD_MS = 160

const games = (() => {
    let list = [], index = 0
    return {
        get index() {
            return index
        },
        get list() {
            return list
        },
        get selected() {
            return list[index].title // selected by the game title, oof
        },
        set index(i) {
            //-2 |
            //-1 | |
            // 0 < | <
            // 1   | |
            // 2 < < |
            //+1 |   |
            //+2 |
            index = i < -1 ? i = 0 :
                i > list.length ? i = list.length - 1 :
                    (i % list.length + list.length) % list.length
        },
        set: (data = []) => list = data.sort((a, b) => a.title.toLowerCase() > b.title.toLowerCase() ? 1 : -1),
        empty: () => list.length === 0
    }
})()

const scroll = ((DEFAULT_INTERVAL) => {
    const state = {
        IDLE: 0, UP: -1, DOWN: 1, DRAG: 3
    }
    let last = state.IDLE
    let _si
    let onShift, onStop

    const shift = (delta) => {
        if (scroll.scrolling) return
        onShift(delta)
        // velocity?
        // keep rolling the game list if the button is pressed
        _si = setInterval(() => onShift(delta), DEFAULT_INTERVAL)
    }

    const stop = () => {
        onStop()
        _si && (clearInterval(_si) && (_si = null))
    }

    const handle = {[state.IDLE]: stop, [state.UP]: shift, [state.DOWN]: shift, [state.DRAG]: null}

    return {
        scroll: (move = state.IDLE) => {
            handle[move] && handle[move](move)
            last = move
        },
        get scrolling() {
            return last !== state.IDLE
        },
        set onShift(fn) {
            onShift = fn
        },
        set onStop(fn) {
            onStop = fn
        },
        state,
        last: () => last
    }
})(SELECT_THRESHOLD_MS)

const ui = (() => {
    const rootEl = document.getElementById('menu-container')
    const choiceMarkerEl = document.getElementById('menu-item-choice')

    const TRANSITION_DEFAULT = `top ${SELECT_THRESHOLD_MS}ms`
    let listTopPos = TOP_POSITION

    rootEl.style.transition = TRANSITION_DEFAULT

    let onTransitionEnd = () => ({})

    //rootEl.addEventListener('transitionend', () => onTransitionEnd())

    let items = []

    const item = (parent) => {
        const title = parent.firstChild.firstChild
        const desc = parent.children[1]

        const _desc = {
            hide: () => gui.hide(desc),
            show: async () => {
                gui.show(desc)
                await gui.anim.fadeIn(desc, .054321)
            },
        }

        const _title = {
            animate: () => title.classList.add('text-move'),
            pick: () => title.classList.add('pick'),
            reset: () => title.classList.remove('pick', 'text-move'),
        }

        const clear = () => {
            _title.reset()
            // _desc.hide()
        }

        return {
            get description() {
                return _desc
            },
            get title() {
                return _title
            },
            clear,
        }
    }

    const render = () => {
        rootEl.innerHTML = games.list.map(game =>
            `<div class="menu-item">` +
            `<div><span>${game.title}</span></div>` +
            //`<div class="menu-item__info">${game.system}</div>` +
            `</div>`)
            .join('')
        items = [...rootEl.querySelectorAll('.menu-item')].map(x => item(x))
    }

    return {
        get items() {
            return items
        },
        get selected() {
            return items[games.index]
        },
        get roundIndex() {
            const closest = Math.round((listTopPos - TOP_POSITION) / -36)
            return closest < 0 ? 0 :
                closest > games.list.length - 1 ? games.list.length - 1 :
                    closest // don't wrap the list on drag
        },
        set onTransitionEnd(x) {
            onTransitionEnd = x
        },
        set pos(idx) {
            listTopPos = TOP_POSITION - idx * 36
            rootEl.style.top = `${listTopPos}px`
        },
        drag: {
            startPos: (pos) => {
                rootEl.style.top = `${listTopPos - pos}px`
                rootEl.style.transition = ''
            },
            stopPos: (pos) => {
                listTopPos -= pos
                rootEl.style.transition = TRANSITION_DEFAULT
            },
        },
        render,
        marker: {
            show: () => gui.show(choiceMarkerEl)
        },
        NO_TRANSITION: onTransitionEnd(),
    }
})(TOP_POSITION, SELECT_THRESHOLD_MS, games)

const show = () => {
    ui.render()
    ui.marker.show() // we show square pseudo-selection marker only after rendering
    scroll.scroll(scroll.state.DOWN) // interactively moves games select down
    scroll.scroll(scroll.state.IDLE)
}

const select = (index) => {
    ui.items.forEach(i => i.clear()) // !to rewrite
    games.index = index
    ui.pos = games.index
}

scroll.onShift = (delta) => select(games.index + delta)

let hasTransition = true // needed for cases when MENU_RELEASE called instead MENU_PRESSED

scroll.onStop = () => {
    const item = ui.selected
    if (item) {
        item.title.pick()
        item.title.animate()
        // hasTransition ? (ui.onTransitionEnd = item.description.show) : item.description.show()
    }
}

sub(MENU_PRESSED, (position) => {
    if (games.empty()) return
    ui.onTransitionEnd = ui.NO_TRANSITION
    hasTransition = false
    scroll.scroll(scroll.state.DRAG)
    ui.selected && ui.selected.clear()
    ui.drag.startPos(position)
})

sub(MENU_RELEASED, (position) => {
    if (games.empty()) return
    ui.drag.stopPos(position)
    select(ui.roundIndex)
    hasTransition = !hasTransition
    scroll.scroll(scroll.state.IDLE)
    hasTransition = true
})

/**
 * Game list module.
 */
export const gameList = {
    scroll: (x) => {
        if (games.empty()) return
        scroll.scroll(x)
    },
    get selected() {
        return games.selected
    },
    set: games.set,
    show: () => {
        if (games.empty()) return
        show()
    },
}
