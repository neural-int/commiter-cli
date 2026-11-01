import Foundation
let state = ProcessInfo.processInfo.thermalState.rawValue
let value: [String: Any] = ["thermal_state": ["nominal", "fair", "serious", "critical"][min(3, max(0, state))], "low_power_mode": ProcessInfo.processInfo.isLowPowerModeEnabled]
print(String(decoding: try JSONSerialization.data(withJSONObject: value, options: [.sortedKeys]), as: UTF8.self))
