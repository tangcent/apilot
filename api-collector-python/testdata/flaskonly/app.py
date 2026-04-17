from flask import Flask

app = Flask(__name__)


@app.route('/status')
def status_handler():
    """statusHandler returns service status."""
    pass
