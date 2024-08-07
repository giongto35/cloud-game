import {
    sub,
    GAME_ROOM_AVAILABLE
} from 'event';

let id = '';

// UI
const roomLabel = document.getElementById('room-txt');

// !to rewrite
const parseURLForRoom = () => {
    let queryDict = {};
    let regex = /^\/?([A-Za-z]*)\/?/g;
    const zone = regex.exec(location.pathname)[1];
    let room = null;

    // get room from URL
    location.search.substr(1)
        .split('&')
        .forEach((item) => {
            queryDict[item.split('=')[0]] = item.split('=')[1]
        });

    if (typeof queryDict.id === 'string') {
        room = decodeURIComponent(queryDict.id);
    }

    return [room, zone];
};

sub(GAME_ROOM_AVAILABLE, data => {
    room.id = data.roomId
    room.save(data.roomId);
}, 1);

/**
 * Game room module.
 */
export const room = {
    get id() {
        return id
    },
    set id(id_) {
        id = id_;
        roomLabel.value = id;
    },
    reset: () => {
        id = '';
        roomLabel.value = id;
    },
    save: (roomIndex) => {
        localStorage.setItem('roomID', roomIndex);
    },
    load: () => localStorage.getItem('roomID'),
    getLink: () => window.location.href.split('?')[0] + `?id=${encodeURIComponent(room.id)}`,
    loadMaybe: () => {
        // localStorage first
        //roomID = loadRoomID();
        let zone = '';

        // Shared URL second
        const [parsedId, czone] = parseURLForRoom();
        if (parsedId !== null) {
            id = parsedId;
        }
        if (czone !== null) {
            zone = czone;
        }

        return [id, zone];
    },
    copyToClipboard: () => {
        const el = document.createElement('textarea');
        el.value = room.getLink();
        document.body.appendChild(el);
        el.select();
        document.execCommand('copy');
        document.body.removeChild(el);
    }
}
