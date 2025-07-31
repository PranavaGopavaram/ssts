# SSTS (System Stress Testing Suite) - Beginner's Guide

## What is SSTS?

SSTS is a **System Stress Testing Suite** - think of it as a tool that puts your computer through various "workouts" to see how well it performs under pressure. Just like how you might test a car by driving it at high speeds or in tough conditions, SSTS tests your computer by making it work really hard with different types of tasks.

## Why Would You Want to Stress Test Your System?

Imagine you're buying a new laptop and want to make sure it won't overheat or crash when you're:
- Running multiple programs at once
- Playing games
- Video editing
- Running servers

SSTS helps you find out:
- **Will my system crash under heavy load?**
- **How hot does my CPU get?**
- **Does my system slow down when memory is full?**
- **Are there any hardware problems I should know about?**

## How Does SSTS Work?

Think of SSTS like a gym for your computer with different "workout stations":

### 1. **The Control Center (Core Orchestrator)**
- This is like a personal trainer that manages all the workouts
- It decides what tests to run and for how long
- It watches over everything to make sure nothing breaks

### 2. **Safety Monitor**
- Like a lifeguard at a pool
- Constantly watches your system's vital signs (CPU temperature, memory usage, etc.)
- If things get too dangerous (like CPU getting too hot), it stops the test immediately

### 3. **Different Types of Stress Tests (Plugins)**
- **CPU Stress**: Makes your processor work at maximum capacity
- **Memory Stress**: Fills up your RAM to see how it handles it
- **Disk Stress**: Tests how fast your storage can read/write data
- **Network Stress**: Tests your internet/network connection

### 4. **Metrics Collector**
- Like a fitness tracker that records everything
- Keeps track of how your system performs during tests
- Records temperature, speed, errors, etc.

### 5. **Web Interface**
- A website you can open in your browser to control everything
- Shows real-time graphs of how your system is doing
- Like a dashboard in your car showing speed, fuel, temperature

## How to Use SSTS (Step by Step)

### Step 1: Installation
```bash
# Download and build the project
git clone <repository-url>
cd ssts
go build
```

### Step 2: Start the System
```bash
# Run the main program
./ssts server
```

This starts a web server that you can access in your browser.

### Step 3: Access the Web Interface
1. Open your web browser
2. Go to `http://localhost:8080` (or whatever port it shows)
3. You'll see a dashboard with options to run different tests

### Step 4: Run Your First Test
1. **Choose a Test Type**: Start with something simple like "CPU Stress"
2. **Set Duration**: Maybe start with 30 seconds for your first test
3. **Set Intensity**: Use 50% to be safe
4. **Click Start**: Watch as your system gets put through its paces

### Step 5: Monitor the Results
While the test runs, you'll see:
- **Real-time graphs** showing CPU usage, temperature, memory usage
- **Safety alerts** if anything gets too hot
- **Performance metrics** showing how well your system handles the load

## Example Scenarios

### Scenario 1: Testing a New Gaming PC
```
Goal: Make sure my new PC can handle intense gaming
Test: Run CPU + GPU stress test for 10 minutes at 80% intensity
Watch for: Temperature staying below 80°C, no crashes
```

### Scenario 2: Server Reliability Test
```
Goal: Ensure my server won't crash under heavy web traffic
Test: Network + Memory stress test for 1 hour
Watch for: Consistent response times, no memory leaks
```

### Scenario 3: Laptop Overheating Check
```
Goal: See if my laptop overheats during intensive work
Test: CPU stress test for 5 minutes, gradually increasing intensity
Watch for: Temperature trends, fan noise, throttling
```

## Understanding the Results

### Good Signs ✅
- Temperature stays within safe limits (usually below 80°C for CPU)
- System remains responsive
- No crashes or freezes
- Performance stays consistent

### Warning Signs ⚠️
- Temperature climbing above 85°C
- System becoming unresponsive
- Performance dropping significantly over time
- Error messages appearing

### Danger Signs 🚨
- Temperature above 95°C
- System crashes or blue screens
- Hardware shutting down automatically
- Burning smells (stop immediately!)

## Safety Features Built-in

SSTS is designed to be safe:

1. **Temperature Monitoring**: Automatically stops if CPU gets too hot
2. **Emergency Stop**: You can always stop tests immediately
3. **Gradual Ramp-up**: Tests start easy and gradually increase intensity
4. **Cooldown Periods**: System waits between intensive tests
5. **Resource Limits**: Won't use more than specified amounts of CPU/memory

## Configuration Files

You can create test configurations like recipes:

```yaml
# cpu-stress-test.yaml
name: "Basic CPU Stress Test"
plugin: "cpu_stress"
duration: "5m"
safety:
  max_cpu_percent: 90
  max_temperature_celsius: 80
config:
  intensity: 70
  cores: 4
```

Then run it with:
```bash
./ssts run-test cpu-stress-test.yaml
```

## Common Use Cases

### For Gamers:
- Test if your PC can handle new games
- Check if cooling is adequate
- Verify system stability for long gaming sessions

### For Developers:
- Test servers before deploying to production
- Validate system performance for applications
- Identify bottlenecks in system resources

### For System Administrators:
- Regular health checks on servers
- Capacity planning (how much load can we handle?)
- Hardware validation after upgrades

### For Hardware Enthusiasts:
- Overclock testing and validation
- Cooling system effectiveness testing
- Component stability verification

## Getting Started Tips

1. **Start Small**: Begin with short, low-intensity tests
2. **Monitor Temperatures**: Always keep an eye on system temperature
3. **Have Good Cooling**: Make sure your fans are working
4. **Save Important Work**: Close important applications before testing
5. **Run Tests When You Don't Need the Computer**: Tests will slow down your system

## Troubleshooting Common Issues

**Q: My system crashed during a test!**
A: This might indicate a hardware problem or inadequate cooling. Start with lower intensity tests.

**Q: The web interface won't load**
A: Make sure the server is running and check the port number in the terminal output.

**Q: Tests are too slow/fast**
A: Adjust the intensity settings in your test configuration.

**Q: I want to stop a test immediately**
A: Click the "Emergency Stop" button in the web interface or press Ctrl+C in the terminal.

## Prerequisites

Before using SSTS, make sure you have:
- Go 1.21 or later installed
- A supported operating system (Linux, macOS, Windows)
- Adequate system cooling
- Administrative privileges (for some system monitoring features)

## Next Steps

Once you're comfortable with basic testing:
1. Explore the API documentation at `/docs` endpoint
2. Create custom test configurations
3. Set up automated testing schedules
4. Integrate with monitoring systems
5. Contribute to plugin development

This stress testing suite helps you understand your system's limits safely and scientifically, just like how doctors use stress tests to check heart health!

---

**⚠️ Important Safety Notice**: Always monitor your system during stress tests. Stop immediately if you notice excessive temperatures, unusual noises, or system instability. SSTS includes safety mechanisms, but hardware varies, and you are responsible for your system's wellbeing.