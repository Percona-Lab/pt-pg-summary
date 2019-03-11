package templates

var TPL = `{{define "report"}}
{{ template "port_and_datadir" .PortAndDatadir }}
{{ template "tablespaces" .Tablespaces }}
{{ if .SlaveHosts96 -}}
  {{ template "slaves_and_lag" .SlaveHosts96 }}
{{ else if .SlaveHosts10 -}}
  {{ template "slaves_and_lag" .SlaveHosts10 }}
{{- end }}
{{ template "cluster" .ClusterInfo }}
{{ template "databases" .Databases }}
{{ template "index_cache_ratios" .IndexCacheHitRatio }}
{{ template "table_cache_ratios" .TableCacheHitRatio }}
{{ template "global_wait_events" .GlobalWaitEvents }}
{{ template "connected_clients" .ConnectedClients }}
{{ template "counters" .Counters }}
{{ end }} {{/* end "report" */}}` +
	`
{{ define "port_and_datadir" -}}
##### --- Database Port and Data_Directory ---- ######
+----------------------+----------------------------------------------------+
|         Name         |                      Setting                       |
+----------------------+----------------------------------------------------+
| {{ printf "%-20s" .Name }} | {{ printf "%-50s" .Setting }} |
+----------------------+----------------------------------------------------+
{{ end -}}
` +
	`{{ define "tablespaces" -}}
##### --- List of Tablespaces ---- ######
+----------------------+----------------------+----------------------------------------------------+
|         Name         |         Owner        |               Location                             |
+----------------------+----------------------+----------------------------------------------------+
{{ range . -}}
| {{ printf "%-20s" .Name }} | {{ printf "%-20s" .Owner }} | {{ printf "%-50s" .Location }} |
{{ end -}}
+----------------------+----------------------+----------------------------------------------------+
{{ end -}} {{/* end define */}}
` +
	`{{ define "slaves_and_lag" -}}
##### --- Slave and the lag with Master ---- ######
{{ if . -}}
+----------------------+----------------------+----------------------------------------------------+
|  Application Name    |    Client Address    |           State                |      Lag          |
+----------------------+----------------------+----------------------------------------------------+
{{ range . -}}
| {{ printf "%-20s" .ApplicationName }} | {{ printf "%-20s" .ClientAddr }} | {{ printf "%-50s" .State }} |  {{ printf "% 4.2f" .ByteLag }}
{{ end -}} {{/* end define */}}
+----------------------+----------------------+----------------------------------------------------+
{{- else -}}
There are no slave hosts
{{ end -}}
{{ end -}}
` +
	`{{ define "cluster" -}}
##### --- Cluster Information ---- ######
{{ if . -}}
+------------------------------------------------------------------------------------------------------+
{{- range . }}                                                                                 
 Usename        : {{ printf "%-20s" .Usename }}                                               
 Time           : {{ printf "%v" .Time }}                                     
 Client Address : {{ printf "%-20s" .ClientAddr }}                            
 Client Hostname: {{ trim .ClientHostname.String 80 }}                        
 Version        : {{ trim .Version 80 }}                                      
 Started        : {{ printf "%v" .Started }}                                  
 Is Slave       : {{ .IsSlave }}                                              
+------------------------------------------------------------------------------------------------------+
{{ end -}}
{{ else -}}
There is no Cluster info
{{ end -}}
{{- end -}} {{/* end define */}}
` +
	`{{ define "databases" -}}
##### --- Databases ---- ######
+----------------------+------------+
|       Dat Name       |    Size    |
+----------------------+------------+
{{ range . -}}
| {{ printf "%-20s" .Datname }} | {{ printf "%10s" .PgSizePretty }} |
{{ end -}}
+----------------------+------------+
{{ end }} {{/* end define */}}
` +
	`{{ define "index_cache_ratios" -}}
##### --- Index Cache Hit Ratios ---- ######
{{ if . -}}
+----------------------+------------+
|      Index Name      |    Ratio   |
+----------------------+------------+
{{ range . -}}
| {{ printf "%-20s" .Name }} |     {{ printf "% 5.2f" .Ratio.Float64 }}  |
{{end -}}
+----------------------+------------+
{{ else -}}
  No stats available
{{ end -}}
{{ end -}} {{/* end define */}}
` +
	`{{ define "table_cache_ratios" -}}
##### --- Table Cache Hit Ratios ---- ######
{{ if . -}}
+----------------------+------------+
|      Index Name      |    Ratio   |
+----------------------+------------+
{{ range . -}}
| {{ printf "%-20s" .Name }} |      {{ printf "%5.2f" .Ratio.Float64 }} |
{{ end -}}
+----------------------+------------+
{{ else -}}
  No stats available
{{ end -}}
{{- end -}} {{/* end define */}}
` +
	`{{ define "global_wait_events" -}}
##### --- List of Wait_events for the entire Cluster - all-databases ---- ######
{{ if . -}}
+----------------------+------------+---------+
|   Wait Event Type    |   Event    |  Count  |
+----------------------+------------+---------+
  {{ range . -}}
  | {{ printf "%-20s" .WaitEventType }} | {{ printf "%5.2f" .WaitEvent }} | {{ printf "% 5d" .Count }} |
  {{- end -}}
+----------------------+------------+
{{ else -}}
  No stats available
{{ end -}}
{{- end -}} {{/* end define */}}
` +
	`{{ define "connected_clients" -}}
##### --- List of users and client_addr or client_hostname connected to --all-databases ---- ######
{{ if . -}}
+----------------------+------------+---------+----------------------+---------+
|   Wait Event Type    |        Client        |         State        |  Count  |
+----------------------+------------+---------+----------------------+---------+
{{ range . -}}` +
	`| {{ printf "%-20s" .Usename }} | ` +
	`{{ printf "%-20s" .Client }} | ` +
	`{{ printf "%-20s" .State }} | ` +
	`{{ printf "% 7d" .Count }} |` + "\n" +
	`{{ end -}}                     
+----------------------+------------+---------+----------------------+---------+
{{ else -}}
  No stats available
{{ end -}}
{{- end -}} {{/* end define */}}
` +
	`{{ define "counters" -}}` +
	"+----------------------" +
	"+-------------" +
	"+------------" +
	"+--------------" +
	"+-------------" +
	"+------------" +
	"+-------------" +
	"+------------" +
	"+-------------" +
	"+------------" +
	"+------------" +
	"+-----------" +
	"+-----------" +
	"+-----------" +
	"+------------+" + "\n" +
	"| Database             " +
	"| Numbackends " +
	"| XactCommit " +
	"| XactRollback " +
	"| BlksRead    " +
	"| BlksHit    " +
	"| TupReturned " +
	"| TupFetched " +
	"| TupInserted " +
	"| TupUpdated " +
	"| TupDeleted " +
	"| Conflicts " +
	"| TempFiles " +
	"| TempBytes " +
	"| Deadlocks  |" + "\n" +
	"+----------------------" +
	"+-------------" +
	"+------------" +
	"+--------------" +
	"+-------------" +
	"+------------" +
	"+-------------" +
	"+------------" +
	"+-------------" +
	"+------------" +
	"+------------" +
	"+-----------" +
	"+-----------" +
	"+-----------" +
	"+------------+" + "\n" +
	`{{ range $key, $value := . -}} ` +
	`| {{ printf "%-20s" (index $value 2).Datname }} ` +
	`| {{ printf "% 7d"  (index $value 2).Numbackends }}     ` +
	`| {{ printf "% 7d"  (index $value 2).XactCommit }}    ` +
	`| {{ printf "% 7d"  (index $value 2).XactRollback }}      ` +
	`| {{ printf "% 7d"  (index $value 2).BlksRead }}     ` +
	`| {{ printf "% 7d"  (index $value 2).BlksHit }}    ` +
	`| {{ printf "% 7d"  (index $value 2).TupReturned }}     ` +
	`| {{ printf "% 7d"  (index $value 2).TupFetched }}    ` +
	`| {{ printf "% 7d"  (index $value 2).TupInserted }}     ` +
	`| {{ printf "% 7d"  (index $value 2).TupUpdated }}    ` +
	`| {{ printf "% 7d"  (index $value 2).TupDeleted }}    ` +
	`| {{ printf "% 7d"  (index $value 2).Conflicts }}   ` +
	`| {{ printf "% 7d"  (index $value 2).TempFiles }}   ` +
	`| {{ printf "% 7d"  (index $value 2).TempBytes }}   ` +
	`| {{ printf "% 7d"  (index $value 2).Deadlocks }}    ` +
	"|\n" +
	`{{ end }}` +
	"+----------------------" +
	"+-------------" +
	"+------------" +
	"+--------------" +
	"+-------------" +
	"+------------" +
	"+-------------" +
	"+------------" +
	"+-------------" +
	"+------------" +
	"+------------" +
	"+-----------" +
	"+-----------" +
	"+-----------" +
	"+------------+" + "\n" +
	`{{ end }}`