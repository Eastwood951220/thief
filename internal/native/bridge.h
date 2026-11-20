#ifndef THIEF_BRIDGE_H
#define THIEF_BRIDGE_H
void thief_run(void);
void thief_apply(const char *json);
void thief_event(char *kind, char *payload);
long thief_fit(const char *text, double width, double height, double font, double spacing, int single);
void thief_chapters(const char *json);
void thief_quit(void);
#endif
