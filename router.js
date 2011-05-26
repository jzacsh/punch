var handlers = require('./views.js').handlers; //@TODO: ref this file in ./router/

//@TODO: place this file in ./router/

//router
var respond = function(response, path, http_POST) {
    console.log('Routing request for %s.', path);
    if (typeof handlers[path] === 'function') {
        handlers[path](response, http_POST);
    }
    else {
        response.writeHead(404, { 'Content-Type': 'text/html' });
        response.write('No page found at "' + path + '".');
        response.end();
    }
}

exports.respond = respond;
