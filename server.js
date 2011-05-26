var http = require('http');
var url = require('url');

var start = function(respond, config) {
    var onRequest = function(req, res) {
        var uri_path = url.parse(req.url).pathname,
            http_POST = '';

        console.log('Receiving request for %s', uri_path);

        req.setEncoding('utf8');
        req.addListener('data', function(http_POST_chunk) {
            http_POST += http_POST_chunk;
        });
        req.addListener('end', function() {
            respond(res, uri_path, http_POST);
        });
    }

    http.createServer(onRequest).listen(config.server.port);
    console.log("Server started.");
}

exports.start = start;
