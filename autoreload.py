import os
import logging
import subprocess
import hmac
from flask import Flask, request, abort, jsonify, render_template_string

app = Flask(__name__)

# configure logging
logging.basicConfig(
    level=logging.DEBUG,
    format='%(asctime)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler()
    ]
)

# count the number of rebuilds
rebuild_count = 0

# Define the HTML template as a string
html_template = """
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>GitHub Webhook Info</title>
    <style>
        body {
            background-color: #121212;
            color: #fff;
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
        }
        
        h1 {
            font-size: 3rem;
            font-weight: 700;
            margin-bottom: 2rem;
            text-align: center;
            text-transform: uppercase;
        }
        
        #info {
            font-size: 2rem;
            margin: 0 auto;
            text-align: center;
        }
        
        #rebuild-count {
            font-size: 4rem;
            font-weight: 700;
            margin-top: 1rem;
        }
    </style>
</head>
<body>
    <h1>KioKen Site Info</h1>
    <div id="info">
        <p>Rebuild Count:</p>
        <p id="rebuild-count">{{ rebuild_count }}</p>
    </div>
</body>
</html>
"""

@app.route('/')
def index():
    global rebuild_count
    rebuild_count_str = str(rebuild_count)
    return render_template_string(html_template, rebuild_count=rebuild_count_str)

@app.route('/webhook', methods=['POST'])
def webhook():
    if request.method == 'POST':
        if 'X-Hub-Signature' not in request.headers:
            abort(400)
        signature = request.headers['X-Hub-Signature']
        sha, signature = signature.split('=')
        if sha != 'sha1':
            abort(501)
        secret_key = os.environ['GITHUB_WEBHOOK_SECRET']
        mac = hmac.new(secret_key.encode(), msg=request.data, digestmod='sha1')
        if not hmac.compare_digest(str(mac.hexdigest()), str(signature)):
            abort(401)
        data = request.get_json()
        if data['action'] == 'push':
            branch = data['ref'].split('/')[-1]
            if branch == 'main':
                logging.info('Received push event on main branch. Updating repository...')
                subprocess.call(['git', 'pull', 'origin', branch])
                logging.info('Repository updated successfully. Rebuilding Go project...')
                subprocess.call(['go', 'build', '-o', 'kioken', 'cmd/kioken/kioken.go'])
                global rebuild_count
                rebuild_count += 1
                logging.info('Go project rebuilt successfully. Restarting the app...')
                pid = subprocess.check_output(['pidof', 'kioken']).decode().strip()
                if pid:
                    subprocess.call(['kill', '-9', pid])
                subprocess.Popen('./kioken')
                logging.info('App restarted successfully.')
        return '', 200
    else:
        abort(400)

@app.route('/info')
def info():
    # get project info
    global rebuild_count
    rebuild_count_str = str(rebuild_count)
    
    return jsonify({'project_info': {'rebuild_count': rebuild_count_str}})

if __name__ == '__main__':
    subprocess.call(['go', 'build', '-o', 'kioken', 'cmd/kioken/kioken.go'])
    subprocess.Popen('./kioken')
    app.run(host='0.0.0.0', port=5000)