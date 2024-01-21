function docReady(callbackFunc) {
    if (document.readyState !== 'loading') {
        callbackFunc();
    } else if (document.addEventListener) {
        document.addEventListener('DOMContentLoaded', callbackFunc);
    } else {
        document.attachEvent('onreadystatechange', function() {
            if (document.readyState === 'complete') {
                callbackFunc();
            }
        });
    }
}

function s2tag(s, isDec) {
    let re = /(?:\.([^.]+))?$/;
    let ext = re.exec(s)[1];
    switch(ext) {
        case "mp4":
            if(isDec) {
                let s2 = s.replace(".mp4", ".webm");
                return '<video controls><source src="' + s2 + '" type="video/webm" /><source src="' + s + '" type="video/mp4" /></video>';
            }else{
                return '<video controls src="'+s+'"></video>';
            }
        case "mp3":
            return '<audio controls loop src="'+s+'"></audio>';
        case "gif":
            return "![](" + s + ")";
        case "jpg":
            return "![](" + s + ")";
        default:
            return s;
    }
}

function postAjax(url, data, success) {
    let xhr = window.XMLHttpRequest ? new XMLHttpRequest() : new ActiveXObject("Microsoft.XMLHTTP");
    xhr.open('POST', url);
    xhr.onreadystatechange = function() {
        if (xhr.readyState>3 && xhr.status===200) { success(xhr.responseText); }
    };
    xhr.setRequestHeader('X-Requested-With', 'XMLHttpRequest');

    if(typeof data === 'string') {
        xhr.setRequestHeader('Content-Type', 'application/json');
        xhr.send(data);
        return xhr;
    }

    xhr.send(data);
    return xhr;
}

function scrollToTop() {
    window.scrollTo({
        top: 0,
        behavior: "smooth"
    });
}