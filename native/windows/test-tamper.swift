import Foundation
import CryptoKit
let root = NSString(string:"~/Downloads/grok-muse/bram").expandingTildeInPath
let peers = try JSONSerialization.jsonObject(with: Data(contentsOf: URL(fileURLWithPath:root+"/peers.json"))) as! [String:[String:Any]]
let identity = try JSONSerialization.jsonObject(with: Data(contentsOf: URL(fileURLWithPath:root+"/machine.json"))) as! [String:String]
let sender = identity["id"]!, secret = peers["win11"]!["secret"] as! String
let message:[String:Any] = ["id":UUID().uuidString,"machine":sender,"from":"TestNoop","to":"TestNoop","created":ISO8601DateFormatter().string(from:Date()),"text":"tamper verification"]
let plain=try JSONSerialization.data(withJSONObject:message)
var combined=try ChaChaPoly.seal(plain, using:SymmetricKey(data:SHA256.hash(data:Data(secret.utf8))), authenticating:Data(("bram/1/message/"+sender).utf8)).combined
combined[combined.count-1] ^= 1
let envelope:[String:Any]=["version":1,"kind":"message","sender":sender,"box":combined.base64EncodedString()]
print(String(data:try JSONSerialization.data(withJSONObject:envelope),encoding:.utf8)!)
