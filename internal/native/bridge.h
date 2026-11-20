#ifndef THIEF_BRIDGE_H
#define THIEF_BRIDGE_H
void thief_run(void);
void thief_apply(const char *json);
void thief_event(char *kind, char *payload);
#endif
