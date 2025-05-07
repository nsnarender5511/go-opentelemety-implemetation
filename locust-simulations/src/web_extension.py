"""
Custom web UI extension for Locust that adds load shape selection.
Task selection is handled by Locust's built-in class picker.
"""
import os
import json
from flask import Flask, request, render_template_string, redirect, jsonify

def init_web_ui_extension(environment):
    """
    Initialize custom web UI extension
    """
    # Skip if we're not running in web UI mode
    if not environment.web_ui:
        return
    
    app = environment.web_ui.app
    
    # Add custom route for getting task execution counts
    @app.route('/task-counts', methods=['GET'])
    def task_counts():
        counts = {}
        # Collect task counts from all user instances
        for user_class, user_instances in environment.runner.user_classes.items():
            if user_class.__name__ == "SimulationUser":
                for user in user_instances:
                    if hasattr(user, "task_counts"):
                        for task, count in user.task_counts.items():
                            if task not in counts:
                                counts[task] = 0
                            counts[task] += count
        
        # Get max limits from command line args
        max_limits = {}
        for task in counts.keys():
            max_name = f"max_{task}"
            if hasattr(environment.parsed_options, max_name):
                max_limits[task] = getattr(environment.parsed_options, max_name)
        
        return jsonify({
            "counts": counts,
            "limits": max_limits
        })
    
    # Add custom route for setting load shape
    @app.route('/load-shape', methods=['GET', 'POST'])
    def load_shape_settings():
        # Get current settings or defaults
        current_load_shape = os.environ.get('LOAD_SHAPE', '')
        
        if request.method == 'POST':
            # Update load shape
            load_shape = request.form.get('load_shape', '')
            os.environ['LOAD_SHAPE'] = load_shape
            
            # Write settings to a file that persists across container restarts
            try:
                with open('/app/settings.env', 'w') as f:
                    f.write(f"LOAD_SHAPE={load_shape}\n")
            except Exception as e:
                # Not critical, but would be nice to have
                pass
            
            # Stop the running test if needed
            # Check if runner is running in a version-compatible way
            try:
                if environment.runner:
                    # Different versions of Locust have different state constants
                    # Try to detect if it's running by checking attributes directly
                    runner_state = getattr(environment.runner, 'state', None)
                    if runner_state is not None:
                        # If we have a state attribute, check against known constants
                        STATE_RUNNING = getattr(environment.runner, 'STATE_RUNNING', 1)
                        STATE_SPAWNING = getattr(environment.runner, 'STATE_SPAWNING', 2)
                        if runner_state in (STATE_RUNNING, STATE_SPAWNING):
                            environment.runner.quit()
                    else:
                        # Older versions may have different ways to check
                        if getattr(environment.runner, 'greenlet', None) is not None:
                            environment.runner.quit()
            except Exception as e:
                # If we can't stop it, just continue
                pass
            
            return """
            <html>
            <head>
                <meta http-equiv="refresh" content="2;url=/" />
                <style>
                    body {
                        font-family: Arial, sans-serif;
                        margin: 20px;
                        background-color: #f5f5f5;
                        text-align: center;
                        padding-top: 100px;
                    }
                    .message {
                        background-color: #4CAF50;
                        color: white;
                        padding: 20px;
                        border-radius: 5px;
                        display: inline-block;
                    }
                </style>
            </head>
            <body>
                <div class="message">
                    <h2>Settings updated!</h2>
                    <p>The new load shape has been applied.</p>
                    <p>Redirecting to the main page in 2 seconds...</p>
                </div>
            </body>
            </html>
            """
        
        # Simple HTML form for settings
        html = """
        <!DOCTYPE html>
        <html>
        <head>
            <title>Locust Load Shape</title>
            <style>
                body {
                    font-family: Arial, sans-serif;
                    margin: 20px;
                    background-color: #f5f5f5;
                }
                .container {
                    max-width: 800px;
                    margin: 0 auto;
                    background-color: white;
                    padding: 20px;
                    border-radius: 5px;
                    box-shadow: 0 2px 5px rgba(0,0,0,0.1);
                }
                h1 {
                    color: #333;
                }
                .form-group {
                    margin-bottom: 15px;
                }
                label {
                    display: block;
                    margin-bottom: 5px;
                    font-weight: bold;
                }
                select, input {
                    width: 100%;
                    padding: 8px;
                    border: 1px solid #ddd;
                    border-radius: 3px;
                }
                button {
                    background-color: #4CAF50;
                    color: white;
                    padding: 10px 15px;
                    border: none;
                    border-radius: 3px;
                    cursor: pointer;
                    font-size: 16px;
                }
                button:hover {
                    background-color: #45a049;
                }
                .section {
                    margin-bottom: 30px;
                    padding-bottom: 20px;
                    border-bottom: 1px solid #eee;
                }
                .note {
                    font-size: 14px;
                    color: #666;
                    margin-top: 5px;
                }
            </style>
        </head>
        <body>
            <div class="container">
                <h1>Load Shape Settings</h1>
                <p>Configure the traffic pattern for your load test.</p>
                
                <form method="POST">
                    <div class="section">
                        <h2>Load Shape</h2>
                        <div class="form-group">
                            <label for="load_shape">Select Load Pattern:</label>
                            <select id="load_shape" name="load_shape">
                                <option value="" """ + ('selected' if current_load_shape == '' else '') + """>Standard (No shape)</option>
                                <option value="stages" """ + ('selected' if current_load_shape == 'stages' else '') + """>Stages (Ramp up, steady, ramp down)</option>
                                <option value="spike" """ + ('selected' if current_load_shape == 'spike' else '') + """>Spike (Sudden traffic surge)</option>
                                <option value="multiple_spikes" """ + ('selected' if current_load_shape == 'multiple_spikes' else '') + """>Multiple Spikes</option>
                                <option value="ramping" """ + ('selected' if current_load_shape == 'ramping' else '') + """>Continuous Ramping</option>
                            </select>
                            <p class="note">Changes the traffic pattern over time</p>
                        </div>
                    </div>
                    
                    <button type="submit">Apply Settings</button>
                    <p class="note">Task weights and enabling/disabling tasks can be configured in the Class Picker</p>
                </form>
                
                <p><a href="/">Back to Locust Dashboard</a></p>
            </div>
        </body>
        </html>
        """
        return render_template_string(html)
    
    # Add a route for task execution counters
    @app.route('/task-execution-counters')
    def task_execution_counters():
        # HTML for a page showing task execution counters
        html = """
        <!DOCTYPE html>
        <html>
        <head>
            <title>Task Execution Counters</title>
            <style>
                body {
                    font-family: Arial, sans-serif;
                    margin: 20px;
                    background-color: #f5f5f5;
                }
                .container {
                    max-width: 800px;
                    margin: 0 auto;
                    background-color: white;
                    padding: 20px;
                    border-radius: 5px;
                    box-shadow: 0 2px 5px rgba(0,0,0,0.1);
                }
                h1 {
                    color: #333;
                }
                table {
                    width: 100%;
                    border-collapse: collapse;
                    margin-top: 20px;
                }
                th, td {
                    padding: 10px;
                    text-align: left;
                    border-bottom: 1px solid #ddd;
                }
                th {
                    background-color: #f2f2f2;
                }
                .progress-container {
                    width: 150px;
                    background-color: #e0e0e0;
                    border-radius: 4px;
                    height: 20px;
                }
                .progress-bar {
                    height: 100%;
                    border-radius: 4px;
                    background-color: #4CAF50;
                }
                .task-name {
                    font-weight: bold;
                }
                .counter {
                    font-weight: bold;
                    font-size: 1.2em;
                }
                .limit {
                    color: #666;
                }
                .refresh-button {
                    background-color: #4CAF50;
                    color: white;
                    padding: 10px 15px;
                    border: none;
                    border-radius: 3px;
                    cursor: pointer;
                    margin-top: 20px;
                }
                .refresh-button:hover {
                    background-color: #45a049;
                }
            </style>
        </head>
        <body>
            <div class="container">
                <h1>Task Execution Counters</h1>
                <p>Current task execution counts and limits</p>
                
                <table id="counters-table">
                    <thead>
                        <tr>
                            <th>Task</th>
                            <th>Count</th>
                            <th>Limit</th>
                            <th>Progress</th>
                        </tr>
                    </thead>
                    <tbody id="counters-body">
                        <tr>
                            <td colspan="4" style="text-align: center;">Loading...</td>
                        </tr>
                    </tbody>
                </table>
                
                <button class="refresh-button" onclick="refreshCounters()">Refresh Counters</button>
                <p><a href="/">Back to Locust Dashboard</a></p>
            </div>
            
            <script>
                // Function to fetch the latest task counts and update the table
                function refreshCounters() {
                    fetch('/task-counts')
                        .then(response => response.json())
                        .then(data => {
                            const tableBody = document.getElementById('counters-body');
                            tableBody.innerHTML = '';
                            
                            // Sort tasks by name
                            const tasks = Object.keys(data.counts).sort();
                            
                            tasks.forEach(task => {
                                const count = data.counts[task];
                                const limit = data.limits[task] || 'Unlimited';
                                
                                const row = document.createElement('tr');
                                
                                // Task name cell
                                const nameCell = document.createElement('td');
                                nameCell.className = 'task-name';
                                nameCell.textContent = task.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase());
                                row.appendChild(nameCell);
                                
                                // Count cell
                                const countCell = document.createElement('td');
                                countCell.className = 'counter';
                                countCell.textContent = count;
                                row.appendChild(countCell);
                                
                                // Limit cell
                                const limitCell = document.createElement('td');
                                limitCell.className = 'limit';
                                limitCell.textContent = limit;
                                row.appendChild(limitCell);
                                
                                // Progress bar cell
                                const progressCell = document.createElement('td');
                                if (limit !== 'Unlimited') {
                                    const progressContainer = document.createElement('div');
                                    progressContainer.className = 'progress-container';
                                    
                                    const progressBar = document.createElement('div');
                                    progressBar.className = 'progress-bar';
                                    const percentage = Math.min(100, (count / limit) * 100);
                                    progressBar.style.width = percentage + '%';
                                    
                                    progressContainer.appendChild(progressBar);
                                    progressCell.appendChild(progressContainer);
                                } else {
                                    progressCell.textContent = 'N/A';
                                }
                                row.appendChild(progressCell);
                                
                                tableBody.appendChild(row);
                            });
                        })
                        .catch(error => {
                            console.error('Error fetching task counts:', error);
                        });
                }
                
                // Refresh counters on page load
                document.addEventListener('DOMContentLoaded', refreshCounters);
                
                // Auto-refresh every 2 seconds
                setInterval(refreshCounters, 2000);
            </script>
        </body>
        </html>
        """
        return render_template_string(html)
    
    # Add links to the custom settings pages in the Locust UI
    @app.context_processor
    def add_custom_settings_links():
        return {
            "custom_settings_links": [
                {"name": "Load Shape", "url": "/load-shape"},
                {"name": "Task Counters", "url": "/task-execution-counters"}
            ]
        }
    
    # Add some CSS to inject our custom menu items
    @app.route('/custom-styles.css')
    def custom_styles():
        return """
        /* Add custom menu link */
        .nav .nav-item {
            display: inline-block;
            margin-right: 10px;
        }
        
        .custom-menu-item {
            display: inline-block;
            padding: 15px;
        }
        
        .custom-menu-item a {
            color: #fff;
            text-decoration: none;
        }
        
        .custom-menu-item a:hover {
            text-decoration: underline;
        }
        """, {'Content-Type': 'text/css'}
    
    # Add a handler for static index.html to inject our custom links
    @app.after_request
    def add_custom_menu_items(response):
        if response.mimetype == 'text/html':
            html = response.get_data(as_text=True)
            if '<ul class="nav navbar-nav">' in html:
                # Add all links
                navbar_html = '<ul class="nav navbar-nav">'
                for link in add_custom_settings_links()["custom_settings_links"]:
                    navbar_html += f'<li class="custom-menu-item"><a href="{link["url"]}">{link["name"]}</a></li>'
                
                # Replace the original navbar
                html = html.replace(
                    '<ul class="nav navbar-nav">',
                    navbar_html
                )
                
                # Add our custom stylesheet
                html = html.replace(
                    '</head>',
                    '<link rel="stylesheet" href="/custom-styles.css"></head>'
                )
                
                response.set_data(html)
        
        return response 