window.addEventListener('load', ()=> {
    var output = document.getElementById('output'),
        input = document.getElementById('input'),
        ws;

    var print = function(message) {
        var d = document.createElement('div');
        d.textContent = message;
        output.appendChild(d);
    };

    document.getElementById('open').addEventListener('click', function(evt) {
        evt.preventDefault();

        if(ws) return false;

        ws = new WebSocket(`ws://${window.location.host}/room/1`);

        ws.onopen = function() {
            print('OPEN');
        };
        ws.onclose = function() {
            print('CLOSE');
            ws = null;
        };
        ws.onmessage = function(evt) {
            print('RESPONSE: ' + evt.data);
        };
        ws.onerror = function(evt) {
            print('ERROR: ' + evt.data);
        };

        return false;
    });

    document.getElementById('send').onclick = function() {
        if(!ws) return false;

        print('SEND: ' + input.value);
        ws.send(JSON.stringify({
            content: {
                text: input.value
            }
        }));
        return false;
    };

    document.getElementById('close').onclick = function() {
        if(!ws) return false;
        ws.close();
        return false;
    };
});
