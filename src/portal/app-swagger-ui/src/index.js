import SwaggerUI from 'swagger-ui'
import 'swagger-ui/dist/swagger-ui.css';

const helpInfo =
    ' If you want to enable basic authorization,' +
    ' please logout Harbor first or manually delete the cookies under the current domain.';
const SAFE_METHODS = ['GET', 'HEAD', 'OPTIONS', 'TRACE'];

function showError(error) {
    const container = document.getElementById('swagger-ui-container');
    if (!container) {
        return;
    }

    container.removeAttribute('class');
    container.innerHTML = '';

    const message = document.createElement('div');
    message.style.margin = '2rem';
    message.style.fontFamily = 'Arial, sans-serif';

    const title = document.createElement('h2');
    title.textContent = 'Failed to load Harbor API documentation';

    const detail = document.createElement('p');
    detail.textContent = error.message;

    const hint = document.createElement('p');
    hint.textContent = 'Please verify that /swagger.json is reachable from this portal.';

    message.appendChild(title);
    message.appendChild(detail);
    message.appendChild(hint);
    container.appendChild(message);
}

// get swagger.json from portal container then render swagger ui
// before rendering, the ui shows a loading style
fetch('/swagger.json').then(value => {
    if (!value.ok) {
        throw new Error(`Request to /swagger.json failed with status ${value.status}`);
    }
    return value.json();
}).then(res => {
    if (!res || !res.info) {
        throw new Error('The response from /swagger.json is not a valid Harbor Swagger document.');
    }

    res['host'] = window.location.host;
    const protocol = window.location.protocol;
    res['schemes'] = [protocol.replace(':', '')];
    res.info.description = res.info.description + helpInfo;
        // start to render
        SwaggerUI({
            spec: res,
            dom_id: '#swagger-ui-container',
            deepLinking: true,
            presets: [SwaggerUI.presets.apis],
            requestInterceptor: request => {
                // Get the csrf token from localstorage
                const token = localStorage.getItem('__csrf');
                const headers = request.headers || {};
                if (token) {
                    if (
                        request.method &&
                        SAFE_METHODS.indexOf(
                            request.method.toUpperCase()
                        ) === -1
                    ) {
                        headers['X-Harbor-CSRF-Token'] = token;
                    }
                }
                return request;
            },
            responseInterceptor: response => {
                const headers = response.headers || {};
                const responseToken =
                    headers['X-Harbor-CSRF-Token'];
                if (responseToken) {
                    // Set the csrf token to localstorage
                    localStorage.setItem('__csrf', responseToken);
                }
                return response;
            },
        });
        // remove loading style
       document.getElementById('swagger-ui-container').removeAttribute('class');

    })
    .catch((err) => {
        console.error(err);
        showError(err);
    });
