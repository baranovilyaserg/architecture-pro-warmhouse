
from flask import Flask, jsonify
import random
from datetime import datetime, timezone

app = Flask(__name__)

@app.route('/temperature/<int:sensor_id>')
def get_temperature(sensor_id):
    value = round(random.uniform(-20, 40), 2)
    response = {
        'value': value,  # float64
        'unit': '°C',    # string
        'timestamp': datetime.now(timezone.utc).isoformat(),  # string (Go time.Time in ISO8601)
        'location': f'sensor_{sensor_id}',  # string
        'status': 'ok',  # string
        'sensor_id': str(sensor_id),  # string
        'sensor_type': 'temperature',  # string
        'description': f'Random temperature for sensor {sensor_id}'  # string
    }
    return jsonify(response)

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=8081)