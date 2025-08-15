## **User Accounts Roles and Permissions**

### **Users**
User accounts in Chariot Server do not model the User, especially in terms of the PI data that is typically managed by OAuth Service applications. The User data needed by Chariot Server is concerned with the OAuth UserID (typically an email), a DisplayName (like a handle) that must be unique, and a collection Roles, which will determine the permissions for a given User.

Value-added attributes like CreatedAt, UpdatedAt, Status, LastLoginAt, etc. may also be added, but perhaps as a 2nd step.

While tempting (and ultimately required) to store User Accounts in one of our supported databases, I think we should leverage our tree saving features instead. Let's admit that Chariot Server as also a database, at least in the sense that it allows secure, multi-user storage of searchable data. We should use Chariot to manage Chariot data first, and reach out to other database technologies in the context of user-defined applications.

Thus, the User Accounts implementation can at first be a Chariot TreeNode agent, similar to the decisionAgent1.json, but serializing a child array JSONNode, that stores the list User Account maps. Roles and Permission list children can be serialized the same way.  Runtime functions for managing the contained data can be added in a rules child.

The goal is a self-contained users agent that can be loaded at startup and used for authentication, but also to power the Chariot internal permissions, etc.

### **Roles**
Typical roles and permissions are along the lines of view, edit, full control, collaborator admin, etc.  We need to go further, by incorporating the Role-Based Function Loading described in the .md file of the same name. As an interpretive language with an inspectable, dynamic runtime, for a built-in library that already has hundreds of functions, Chariot needs to break new ground in security and efficiency. Role-Based Function (and Agent) loading provides that extra support.

1. A Role contains the list of Chariot built-in functions that will be loaded at Session creation. Every Session gets it's own Runtime, so every Session Runtime can load a subset of built-in functions, provided by the Role.

2. User-Defined Functions (UDFs) can also be controlled by a whitelist of UDFs in the Role. However, please note that this creates a dependency on the function library file configured for load on Session creation.

3. A list of load-on-start Agents can control what system agents become present.

4. Each entry in the role function and agent lists need to have a permission flag

	- View -- can see and inspect the objects, but cannot modify or delete them

	- Contribute -- typically given to developers who are writing and testing agents and UDFs

	- Execute -- in a published agent, only Execute permissions need to exist for the internal system account

### Storage
When the User Agent TreeNode is saved, it needs to be stored in encrypted GOB format.

## Container Image Publishing
There is also .md file about Containerizing a Chariot Agent. As part of that process:

1. The Roles should be configured to load only the Chariot built-ins, UDFs and agents required. Thus, a decision engine agent probably has no need for advanced math and trig, but may well need some of the more specialized business financial functions.