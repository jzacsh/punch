var querystring = require('querystring');
var punch = require('./punch.js'); //@TODO: ref this file in ../

//@TODO: place this file in ./router/

//@TODO: define an api here, to avoid repeatedcode of the below req/resp handling.

//routes
var handlers = []
handlers['/'] = handlers['/start'] = function(response, http_POST) {
    console.log('request handler for "start" was called.');
    response.writeHead(200, { 'Content-Type': 'text/html' });
    response.write('hello from start handler.');
    response.end();
};
handlers['/stop'] = function(response, http_POST) {
    console.log('request handler for "stop" was called.');
    response.writeHead(200, { 'Content-Type': 'text/html' });
    response.write('hello from stop handler.');
    var note = querystring.parse(http_POST)['note'];
    if (typeof note !== 'undefined') {
        response.write('Received note: ' + note);
    }
    response.end();
};
handlers['/403'] = function(response, http_POST) {
    console.log('request handler for "403" was called.');
    response.writeHead(403, { 'Content-Type': 'text/html' });
    response.write('Access denied. Please authenticate, first.');
    response.end();
};

exports.handlers = handlers
