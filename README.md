smart-meter-exporter
--------------------

prometheus exporter voor een smartmeter, met P1 kabel

Ik heb een P1/USB kabel, ik kan de stroomverbruikt en teruglevering van mijn
smartmeters lezen.

Vanwege deze prometheus exporter wordt de stroomverbruikt van mijn huis op
kaart gebracht

```
    │ │ │ │ │ │
    │ │ │ │ │ │
    │ │ │ │ │ │
   ┌┴─┴─┴─┴─┴─┴┐
   │           │
   │groepenkast│
   │           │
   │           │
   │           │           Prometheus
   │           │           Remote
   │           │           Write
   │           │              ▲
   │           │              │
   │           │              │
   └─────┬─────┘              │
         │                    │
         │                    │
    ┌────┴────┐ P1         ┌──┴─┐
    │         │◄───────────┤RPi │
    │ smart   │        USB │    │
    │   meter │            └────┘
    │         │
    └────┬────┘
         │
         │
         │ stroomnet
         │
         ▼

```

Ik heb deze codes alleen met een 1-fase meter getest

Lees de handleiding voor P1 poorten [hier](https://domoticx.com/p1-poort-slimme-meter-hardware/)
