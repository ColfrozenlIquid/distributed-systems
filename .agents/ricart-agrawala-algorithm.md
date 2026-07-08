The **Ricart--Agrawala algorithm** is an algorithm for mutual exclusion on a distributed system.
This algorithm is an extension and optimization of Lamport\'s Distributed Mutual Exclusion Algorithm,
by removing the need for release messages.[^1] It was developed by
computer scientists [Glenn
Ricart](Glenn_Ricart "Glenn Ricart"){.wikilink} and [Ashok
Agrawala](Ashok_Agrawala "Ashok Agrawala"){.wikilink}.
 
## Algorithm
 
### Terminology
 
- A *site* is any computing device which runs the Ricart-Agrawala
  Algorithm
- The *requesting site* is the site which is requesting to enter the
  critical section.
- The *receiving site* is every other site which is receiving a request
  from the requesting site.
### Algorithm {#algorithm_1}
 
**Requesting Site**
 
- Sends a message to all sites. This message includes the site\'s name,
  and the current timestamp of the system according to its logical
  clock (*which is assumed to be synchronized with the other sites*)
**Receiving Site**
 
- Upon reception of a request message, immediately sending a timestamped
  *reply* message if and only if:
:\* the receiving process is not currently interested in the critical
section OR
 
:\* the receiving process has a lower priority (*usually this means
having a later timestamp*)
 
- Otherwise, the receiving process will defer the reply message. This
  means that a reply will be sent only after the receiving process has
  finished using the critical section itself.
**Critical Section:**
 
- Requesting site enters its critical section only after receiving all
  reply messages.
- Upon exiting the critical section, the site sends all deferred reply
  messages.
### Performance
 
- Max number of network messages: $2*(N-1)$
- Synchronization Delays: One message propagation delay
### Common optimizations {#common_optimizations}
 
Once site $P_i$ has received a $reply$ message from site $P_j$, site
$P_i$ may enter the critical section multiple times without receiving
permission from $P_j$ on subsequent attempts up to the moment when $P_i$
has sent a $reply$ message to $P_j$. This is called Roucairol-Carvalho
optimization or Roucairol-Carvalho algorithm.