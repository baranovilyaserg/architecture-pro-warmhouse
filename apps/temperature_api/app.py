
from flask import Flask, request, jsonify
import random
from datetime import datetime, timezone

app = Flask(__name__)

@app.route('/temperature')
def get_temperature():
    location = request.args.get('location', 'unknown')
    sensor_id = request.args.get('sensorId', location)
    value = round(random.uniform(-20, 40), 2)
    response = {
        'value': value,
        'unit': 'C',
        'timestamp': datetime.now(timezone.utc).isoformat(),
        'location': location,
        'status': 'ok',
        'sensor_id': sensor_id,
        'sensor_type': 'temperature',
        'description': f'Random temperature for {location}'
    }
    return jsonify(response)

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=8081)