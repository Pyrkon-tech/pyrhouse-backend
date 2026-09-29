-- Fixtures: anonymised data from staging-pyrhouse-2026-08-06_233419-restore.sql, built by postgres/build-fixtures.sh on 2026-09-28.
-- Load into a migrated database; it replaces whatever the tables hold. Passwords: "pyrhouse".
TRUNCATE app_settings, audit_logs, equipment_request_category_mapping, equipment_request_items, equipment_request_location_mapping, equipment_request_price_list, equipment_request_quests, equipment_request_sync_log, item_category, items, locations, non_serialized_items, non_serialized_transfers, origins, pyr_code_reservations, quest_transfers, release_assets, release_stocks, releases, schedule_assignments, schedule_day_windows, schedule_slots, schedule_volunteers, schedules, serialized_transfers, service_desk_request_comments, service_desk_requests, transfer_users, transfers, users RESTART IDENTITY CASCADE;
--
-- PostgreSQL database dump
--

\restrict fhfvY2tZyL43oPSniNxyy5rL0KXQ0gwYZr2sCcFRlW4uynHHNzPREFgPp4gJfzq

-- Dumped from database version 16.14 (Debian 16.14-1.pgdg13+1)
-- Dumped by pg_dump version 16.14 (Debian 16.14-1.pgdg13+1)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Data for Name: app_settings; Type: TABLE DATA; Schema: public; Owner: -
--

SET SESSION AUTHORIZATION DEFAULT;

ALTER TABLE public.app_settings DISABLE TRIGGER ALL;

COPY public.app_settings (key, value, description, updated_at) FROM stdin;
equipment_request.sheet_name	Arkusz1	Sheet tab name within the document	2026-05-11 10:03:01.348079
scheduling.sheet_name	Grafik	Schedule - Sheet tab name	2026-05-12 07:45:48.43208
equipment_request.cennik_sheet_name	Cennik	Sheet name for price list (Cennik tab)	2026-05-14 20:27:09.516847
equipment_request.sheet_id	fixture-sheet-id	Google Sheets document ID for equipment requests	2026-05-11 09:44:08.106183
scheduling.sheet_id	fixture-sheet-id	Schedule - Google Sheets document ID	2026-05-12 07:50:39.609197
\.


ALTER TABLE public.app_settings ENABLE TRIGGER ALL;

--
-- Data for Name: users; Type: TABLE DATA; Schema: public; Owner: -
--

ALTER TABLE public.users DISABLE TRIGGER ALL;

COPY public.users (id, username, fullname, password_hash, role, points, active, discord_id, discord_username, avatar_url, auth_provider, google_id, google_email) FROM stdin;
105	user105	Agnieszka Lewandowski	$2a$10$pZ4nchhrTGo5/UHuZGe0v.PB1OHGxPz/pX3V.7JRXhFpZKv10eCpu	user	0	t	100000000000000105	discord_user105	\N	discord	\N	\N
13	user13	Kamil Król	$2a$10$Zp.TM8zyjtm4FzYPM/1JL.WVa2WJBwmmmQax/alCmqQXocDDKg4rS	moderator	10	t	\N	\N	\N	local	\N	\N
110	user110	Tomasz Król	$2a$10$jg.f.zeH6V3SnK6XDZt0leVZfUTCOuDpvTDJCwpyOQpHe4CJdIHbO	user	0	t	100000000000000110	discord_user110	\N	discord	\N	\N
108	user108	Zofia Kwiatkowski	$2a$10$Dxi8a5r09/JfNv8mROVlm./Rv5SQd1O3wl/cJ5wM136pmaFO4htkC	user	0	t	100000000000000108	discord_user108	\N	discord	\N	\N
114	user114	Jan Wiśniewski	$2a$10$9cqrqq91kaz.CMhySlxPo.sovVLgSH7OfvHNpNLN37dUIU9BmL2Z.	user	0	t	\N	\N	\N	local	\N	\N
23	user23	Michał Kwiatkowski	$2a$10$S1hkPXP1BJj9gWFRQ0.qDerPh06S9j5mrgwt2/2EKxTTEyJ2HeyES	dispatcher	0	t	100000000000000023	discord_user23	\N	local	\N	\N
11	user11	Wojciech Zieliński	$2a$10$kw4MGsr.HVEFVajcCPlIZuL.1HzGc6/r6Kd/jtWOkH3cj9TjrNt/6	admin	10	t	100000000000000011	discord_user11	\N	local	google-11	user11@example.com
112	user112	Piotr Woźniak	$2a$10$PPYzzdenDbkG6gABxeU/g.T5v4qKYndjRfF8/SEn0uuJlPKHsWfx.	moderator	0	t	100000000000000112	discord_user112	\N	discord	\N	\N
2	user2	Natalia Wiśniewski	$2a$10$B11h.6dEEwxtq5yFBGb/lura.8Ku83FiwRem2asmI40BnxCJW7Nqy	moderator	0	t	\N	\N	\N	local	\N	\N
9	user9	Agnieszka Lewandowski	$2a$10$l.QJ0eW8bN4fOzV211ooH.z1xLr89.G/U9TSSDWfzNUL52GRnJXbu	admin	0	t	100000000000000009	discord_user9	\N	google	google-9	user9@example.com
8	user8	Kamil Jankowski	$2a$10$T5b78m2z/Z235ySXgkHbA.7955HMvdjHzBLVCUoacmHwGlUUzBo.q	moderator	0	t	100000000000000008	discord_user8	\N	local	\N	\N
3	admin	Anna Mazur	$2a$10$s1C8DPZ/14sZAueM3DA3QeAj5qYAFAE/LuDB.JuLR//3GdlbOdRXa	admin	0	t	\N	\N	\N	local	\N	\N
10	user10	Wojciech Krawczyk	$2a$10$w8dfeLVvY0nOlw8GwWzoJ.LvjzcSW7J8wpGhvE.qt/Hzf6bNUXEj6	moderator	0	t	100000000000000010	discord_user10	\N	local	\N	\N
1	user1	Tomasz Krawczyk	$2a$10$QpD9lw0S.DjXtup6WCJp6.CdzkkjlIAtSrMfwm0qo69BeiML2Y1fa	dispatcher	0	f	\N	\N	\N	local	\N	\N
106	user106	Julia Kwiatkowski	$2a$10$EEwLruL8QwEGeUgmyFnScOtvRiGQVAdQCkoECVVndejs4VZDqiPx2	user	0	t	\N	\N	\N	local	\N	\N
111	user111	Tomasz Kozłowski	$2a$10$E9aOZtLpCc192QDgDCZfAuk0cxI2VUrNZWcxocL8Ml/jujiEQurW.	user	0	t	100000000000000111	discord_user111	\N	discord	\N	\N
113	user113	Łukasz Mazur	$2a$10$A52Qd0QJsJUctrLaWtMhPeAzPsAA5pdE.z.jEp5WrVXvxtHByTXbC	admin	0	t	\N	\N	\N	google	google-113	user113@example.com
65	user65	Maria Kwiatkowski	$2a$10$OU2VuisshcQLiVGccm2vpe7S4.ObyZzvvJtSmh42KS351u6M47PWO	user	0	t	100000000000000065	discord_user65	\N	discord	\N	\N
67	user67	Ewa Pawłowski	$2a$10$UeHpdCYpORPJ6bDXZY5wmu.vKpIpyc3aRHosBVn/vmtDjypwxobyO	user	0	t	100000000000000067	discord_user67	\N	discord	\N	\N
109	user109	Krzysztof Dąbrowski	$2a$10$Oo1wGANNp4fjPuk354S57uoNwDr7mexFckuKtHbEQzmvztpiHmiH.	user	0	t	100000000000000109	discord_user109	\N	discord	\N	\N
72	user72	Katarzyna Zieliński	$2a$10$tk8EQ1TBokO.piPGY8YivulXb.2HLMVs9C/23KKGjfw7Hw0rW.WwC	user	0	t	100000000000000072	discord_user72	\N	discord	\N	\N
68	user68	Michał Woźniak	$2a$10$77NHylo7wY4eiDIIEHs0o.O9URLcEjpce89QhWqQnf70yMasyu7JG	user	0	t	100000000000000068	discord_user68	\N	discord	\N	\N
70	user70	Łukasz Dąbrowski	$2a$10$1/Ks/ILHc0HnSOZGjFSpB.UFwdOD0VXZi/a82nCPiOfCWkeUBfvcu	user	0	t	100000000000000070	discord_user70	\N	discord	\N	\N
66	user66	Ewa Król	$2a$10$GITyuxHkdNfOgNgykHLA5.71nHQmthgvD9wO2j4OCanBWJezoOLWe	user	0	t	100000000000000066	discord_user66	\N	discord	\N	\N
69	user69	Michał Grabowski	$2a$10$FxKicV252w6FJP4ynFtpq.9hL9BgF87.qPGf5T5T63mWSR2dMAGYS	user	0	t	100000000000000069	discord_user69	\N	discord	\N	\N
71	user71	Wojciech Lewandowski	$2a$10$iATLLl1sb3ucKOnts021iegTZG5DUSrgB5htTWxo6JDX3RWzQbH6q	user	0	t	\N	\N	\N	local	\N	\N
\.


ALTER TABLE public.users ENABLE TRIGGER ALL;

--
-- Data for Name: audit_logs; Type: TABLE DATA; Schema: public; Owner: -
--

ALTER TABLE public.audit_logs DISABLE TRIGGER ALL;

COPY public.audit_logs (id, resource_id, resource_type, action, data, created_at, user_id) FROM stdin;
2200	247	transfer	in_transfer	{}	2026-06-21 18:54:07.554783	\N
2061	229	transfer	delivered	{}	2026-06-21 17:50:53.156012	\N
2062	267	asset	delivered	{}	2026-06-21 17:50:53.157959	\N
2063	266	asset	delivered	{}	2026-06-21 17:50:53.159199	\N
2064	265	asset	delivered	{}	2026-06-21 17:50:53.160299	\N
2065	242	asset	delivered	{}	2026-06-21 17:50:53.161473	\N
2066	241	asset	delivered	{}	2026-06-21 17:50:53.162675	\N
2067	240	asset	delivered	{}	2026-06-21 17:50:53.163718	\N
2068	93	asset	delivered	{}	2026-06-21 17:50:53.164775	\N
2069	71	asset	delivered	{}	2026-06-21 17:50:53.165989	\N
2070	70	asset	delivered	{}	2026-06-21 17:50:53.167118	\N
2071	69	asset	delivered	{}	2026-06-21 17:50:53.16848	\N
2072	68	asset	delivered	{}	2026-06-21 17:50:53.169404	\N
2073	67	asset	delivered	{}	2026-06-21 17:50:53.170345	\N
2074	66	asset	delivered	{}	2026-06-21 17:50:53.171314	\N
2075	64	asset	delivered	{}	2026-06-21 17:50:53.172514	\N
2076	39	asset	delivered	{}	2026-06-21 17:50:53.173362	\N
2077	5	asset	delivered	{}	2026-06-21 17:50:53.174284	\N
2078	2	stock	delivered	{}	2026-06-21 17:50:53.175166	\N
2079	65	stock	delivered	{}	2026-06-21 17:50:53.176194	\N
2080	66	stock	delivered	{}	2026-06-21 17:50:53.177167	\N
2081	230	transfer	in_transfer	{}	2026-06-21 17:59:30.151139	\N
2082	105	asset	in_transfer	{}	2026-06-21 17:59:30.15782	\N
2083	230	transfer	delivered	{}	2026-06-21 17:59:34.713737	\N
2084	105	asset	delivered	{}	2026-06-21 17:59:34.721981	\N
2085	231	transfer	in_transfer	{}	2026-06-21 18:06:57.17473	\N
2086	298	asset	in_transfer	{}	2026-06-21 18:06:57.176632	\N
2087	112	stock	in_transfer	{}	2026-06-21 18:06:57.177937	\N
2088	231	transfer	delivered	{}	2026-06-21 18:06:59.343527	\N
2089	298	asset	delivered	{}	2026-06-21 18:06:59.344541	\N
2090	112	stock	delivered	{}	2026-06-21 18:06:59.345474	\N
2091	232	transfer	in_transfer	{}	2026-06-21 18:10:42.470196	\N
2092	107	asset	in_transfer	{}	2026-06-21 18:10:42.471769	\N
2093	232	transfer	delivered	{}	2026-06-21 18:10:44.347306	\N
2094	107	asset	delivered	{}	2026-06-21 18:10:44.350196	\N
2095	233	transfer	in_transfer	{}	2026-06-21 18:12:12.543618	\N
2096	120	asset	in_transfer	{}	2026-06-21 18:12:12.545878	\N
2097	233	transfer	delivered	{}	2026-06-21 18:12:14.335016	\N
2098	120	asset	delivered	{}	2026-06-21 18:12:14.336223	\N
2099	234	transfer	in_transfer	{}	2026-06-21 18:13:33.427693	\N
2100	104	asset	in_transfer	{}	2026-06-21 18:13:33.429526	\N
2101	234	transfer	delivered	{}	2026-06-21 18:13:35.624194	\N
2102	104	asset	delivered	{}	2026-06-21 18:13:35.625357	\N
2103	235	transfer	in_transfer	{}	2026-06-21 18:20:26.806741	\N
2104	8	asset	in_transfer	{}	2026-06-21 18:20:26.808242	\N
2105	47	asset	in_transfer	{}	2026-06-21 18:20:26.809436	\N
2106	48	asset	in_transfer	{}	2026-06-21 18:20:26.81076	\N
2107	10	asset	in_transfer	{}	2026-06-21 18:20:26.812007	\N
2108	11	asset	in_transfer	{}	2026-06-21 18:20:26.813761	\N
2109	12	asset	in_transfer	{}	2026-06-21 18:20:26.815091	\N
2110	13	asset	in_transfer	{}	2026-06-21 18:20:26.816582	\N
2111	235	transfer	delivered	{}	2026-06-21 18:20:28.334619	\N
2112	8	asset	delivered	{}	2026-06-21 18:20:28.339982	\N
2113	47	asset	delivered	{}	2026-06-21 18:20:28.341884	\N
2114	48	asset	delivered	{}	2026-06-21 18:20:28.348431	\N
2115	10	asset	delivered	{}	2026-06-21 18:20:28.352487	\N
2116	11	asset	delivered	{}	2026-06-21 18:20:28.356304	\N
2117	12	asset	delivered	{}	2026-06-21 18:20:28.357754	\N
2118	13	asset	delivered	{}	2026-06-21 18:20:28.359288	\N
2119	236	transfer	in_transfer	{}	2026-06-21 18:23:44.3747	\N
2120	184	asset	in_transfer	{}	2026-06-21 18:23:44.377353	\N
2121	189	asset	in_transfer	{}	2026-06-21 18:23:44.378759	\N
2122	108	asset	in_transfer	{}	2026-06-21 18:23:44.380319	\N
2123	87	asset	in_transfer	{}	2026-06-21 18:23:44.381402	\N
2124	146	asset	in_transfer	{}	2026-06-21 18:23:44.382806	\N
2125	147	asset	in_transfer	{}	2026-06-21 18:23:44.384	\N
2126	148	asset	in_transfer	{}	2026-06-21 18:23:44.385345	\N
2127	164	asset	in_transfer	{}	2026-06-21 18:23:44.386593	\N
2128	296	asset	in_transfer	{}	2026-06-21 18:23:44.387739	\N
2129	236	transfer	delivered	{}	2026-06-21 18:23:49.005948	\N
2130	184	asset	delivered	{}	2026-06-21 18:23:49.007287	\N
2131	189	asset	delivered	{}	2026-06-21 18:23:49.008323	\N
2132	108	asset	delivered	{}	2026-06-21 18:23:49.009588	\N
2133	87	asset	delivered	{}	2026-06-21 18:23:49.010993	\N
2134	146	asset	delivered	{}	2026-06-21 18:23:49.012314	\N
2135	147	asset	delivered	{}	2026-06-21 18:23:49.013536	\N
2136	148	asset	delivered	{}	2026-06-21 18:23:49.014885	\N
2137	164	asset	delivered	{}	2026-06-21 18:23:49.016372	\N
2138	296	asset	delivered	{}	2026-06-21 18:23:49.017458	\N
2139	237	transfer	in_transfer	{}	2026-06-21 18:24:08.390741	\N
2140	9	stock	in_transfer	{}	2026-06-21 18:24:08.392423	\N
2141	237	transfer	delivered	{}	2026-06-21 18:24:09.847802	\N
2142	9	stock	delivered	{}	2026-06-21 18:24:09.849073	\N
2143	238	transfer	in_transfer	{}	2026-06-21 18:25:45.666486	\N
2144	285	asset	in_transfer	{}	2026-06-21 18:25:45.668275	\N
2145	238	transfer	delivered	{}	2026-06-21 18:25:46.967718	\N
2146	285	asset	delivered	{}	2026-06-21 18:25:46.969927	\N
2147	239	transfer	in_transfer	{}	2026-06-21 18:26:08.566131	\N
2148	172	asset	in_transfer	{}	2026-06-21 18:26:08.568339	\N
2149	171	asset	in_transfer	{}	2026-06-21 18:26:08.570152	\N
2150	239	transfer	delivered	{}	2026-06-21 18:29:47.420243	\N
2151	172	asset	delivered	{}	2026-06-21 18:29:47.422159	\N
2152	171	asset	delivered	{}	2026-06-21 18:29:47.423858	\N
2153	240	transfer	in_transfer	{}	2026-06-21 18:45:32.939719	\N
2154	81	asset	in_transfer	{}	2026-06-21 18:45:32.943853	\N
2155	119	asset	in_transfer	{}	2026-06-21 18:45:32.945719	\N
2156	40	asset	in_transfer	{}	2026-06-21 18:45:32.947686	\N
2157	41	asset	in_transfer	{}	2026-06-21 18:45:32.950577	\N
2158	15	stock	in_transfer	{}	2026-06-21 18:45:32.951688	\N
2159	240	transfer	delivered	{}	2026-06-21 18:45:34.424655	\N
2160	81	asset	delivered	{}	2026-06-21 18:45:34.425894	\N
2161	119	asset	delivered	{}	2026-06-21 18:45:34.427011	\N
2162	40	asset	delivered	{}	2026-06-21 18:45:34.428334	\N
2163	41	asset	delivered	{}	2026-06-21 18:45:34.429775	\N
2164	15	stock	delivered	{}	2026-06-21 18:45:34.430902	\N
2165	241	transfer	in_transfer	{}	2026-06-21 18:45:43.32553	\N
2166	183	asset	in_transfer	{}	2026-06-21 18:45:43.327226	\N
2167	63	asset	in_transfer	{}	2026-06-21 18:45:43.329943	\N
2168	65	asset	in_transfer	{}	2026-06-21 18:45:43.331982	\N
2169	244	asset	in_transfer	{}	2026-06-21 18:45:43.334198	\N
2170	204	asset	in_transfer	{}	2026-06-21 18:45:43.336002	\N
2171	2	stock	in_transfer	{}	2026-06-21 18:45:43.337415	\N
2172	241	transfer	delivered	{}	2026-06-21 18:45:48.194338	\N
2173	183	asset	delivered	{}	2026-06-21 18:45:48.195969	\N
2174	63	asset	delivered	{}	2026-06-21 18:45:48.197279	\N
2175	65	asset	delivered	{}	2026-06-21 18:45:48.198243	\N
2176	244	asset	delivered	{}	2026-06-21 18:45:48.199114	\N
2177	204	asset	delivered	{}	2026-06-21 18:45:48.200209	\N
2178	2	stock	delivered	{}	2026-06-21 18:45:48.201298	\N
2179	242	transfer	in_transfer	{}	2026-06-21 18:46:04.833336	\N
2180	15	stock	in_transfer	{}	2026-06-21 18:46:04.835014	\N
2181	242	transfer	delivered	{}	2026-06-21 18:46:06.591626	\N
2182	15	stock	delivered	{}	2026-06-21 18:46:06.592755	\N
2183	243	transfer	in_transfer	{}	2026-06-21 18:49:33.542786	\N
2184	190	asset	in_transfer	{}	2026-06-21 18:49:33.544075	\N
2185	104	stock	in_transfer	{}	2026-06-21 18:49:33.545259	\N
2186	244	transfer	in_transfer	{}	2026-06-21 18:51:03.800771	\N
2187	79	asset	in_transfer	{}	2026-06-21 18:51:03.802466	\N
2188	244	transfer	delivered	{}	2026-06-21 18:51:05.303732	\N
2189	79	asset	delivered	{}	2026-06-21 18:51:05.305498	\N
2190	245	transfer	in_transfer	{}	2026-06-21 18:52:55.200126	\N
2191	226	asset	in_transfer	{}	2026-06-21 18:52:55.202708	\N
2192	245	transfer	delivered	{}	2026-06-21 18:52:56.665796	\N
2193	226	asset	delivered	{}	2026-06-21 18:52:56.667144	\N
2194	246	transfer	in_transfer	{}	2026-06-21 18:53:19.106165	\N
2195	165	asset	in_transfer	{}	2026-06-21 18:53:19.107087	\N
2196	22	stock	in_transfer	{}	2026-06-21 18:53:19.10803	\N
2197	246	transfer	delivered	{}	2026-06-21 18:53:22.220696	\N
2198	165	asset	delivered	{}	2026-06-21 18:53:22.222697	\N
2199	22	stock	delivered	{}	2026-06-21 18:53:22.22372	\N
2201	126	asset	in_transfer	{}	2026-06-21 18:54:07.556258	\N
2202	127	asset	in_transfer	{}	2026-06-21 18:54:07.558398	\N
2203	20	stock	in_transfer	{}	2026-06-21 18:54:07.559586	\N
2204	247	transfer	delivered	{}	2026-06-21 18:54:10.439786	\N
2205	126	asset	delivered	{}	2026-06-21 18:54:10.441197	\N
2206	127	asset	delivered	{}	2026-06-21 18:54:10.452184	\N
2207	20	stock	delivered	{}	2026-06-21 18:54:10.454773	\N
2208	248	transfer	in_transfer	{}	2026-06-21 18:55:13.639052	\N
2209	97	asset	in_transfer	{}	2026-06-21 18:55:13.641296	\N
2210	175	asset	in_transfer	{}	2026-06-21 18:55:13.642529	\N
2211	176	asset	in_transfer	{}	2026-06-21 18:55:13.643589	\N
2212	248	transfer	delivered	{}	2026-06-21 18:55:17.367053	\N
2213	97	asset	delivered	{}	2026-06-21 18:55:17.368165	\N
2214	175	asset	delivered	{}	2026-06-21 18:55:17.369247	\N
2215	176	asset	delivered	{}	2026-06-21 18:55:17.370331	\N
2216	249	transfer	in_transfer	{}	2026-06-21 19:31:17.946062	\N
2217	177	asset	in_transfer	{}	2026-06-21 19:31:17.947967	\N
2218	250	transfer	in_transfer	{}	2026-06-21 19:39:30.589782	\N
2219	186	asset	in_transfer	{}	2026-06-21 19:39:30.594576	\N
2220	250	transfer	delivered	{}	2026-06-21 19:39:34.784489	\N
2221	186	asset	delivered	{}	2026-06-21 19:39:34.786043	\N
2222	251	transfer	in_transfer	{}	2026-06-21 19:41:36.144235	\N
2223	109	asset	in_transfer	{}	2026-06-21 19:41:36.145772	\N
2224	251	transfer	delivered	{}	2026-06-21 19:41:37.745233	\N
2225	109	asset	delivered	{}	2026-06-21 19:41:37.74637	\N
2226	243	transfer	delivered	{}	2026-06-21 19:43:18.941806	\N
2227	190	asset	delivered	{}	2026-06-21 19:43:18.942867	\N
2228	104	stock	delivered	{}	2026-06-21 19:43:18.943983	\N
2229	252	transfer	in_transfer	{}	2026-06-22 08:59:45.883138	\N
2230	7	asset	in_transfer	{}	2026-06-22 08:59:45.885656	\N
2231	6	asset	in_transfer	{}	2026-06-22 08:59:45.886996	\N
2232	252	transfer	delivered	{}	2026-06-22 08:59:47.658434	\N
2233	7	asset	delivered	{}	2026-06-22 08:59:47.659988	\N
2234	6	asset	delivered	{}	2026-06-22 08:59:47.66205	\N
2235	253	transfer	in_transfer	{}	2026-06-22 09:44:25.333402	\N
2236	43	asset	in_transfer	{}	2026-06-22 09:44:25.337966	\N
2237	253	transfer	delivered	{}	2026-06-22 09:44:26.626261	\N
2238	43	asset	delivered	{}	2026-06-22 09:44:26.627552	\N
2239	254	transfer	in_transfer	{}	2026-06-22 09:44:52.036658	\N
2240	42	asset	in_transfer	{}	2026-06-22 09:44:52.038049	\N
2241	254	transfer	delivered	{}	2026-06-22 09:44:53.748103	\N
2242	42	asset	delivered	{}	2026-06-22 09:44:53.74952	\N
2243	249	transfer	delivered	{}	2026-06-22 10:09:25.322725	\N
2244	177	asset	delivered	{}	2026-06-22 10:09:25.32644	\N
2245	255	transfer	in_transfer	{}	2026-06-22 10:09:59.408884	\N
2246	144	asset	in_transfer	{}	2026-06-22 10:09:59.410197	\N
2247	149	asset	in_transfer	{}	2026-06-22 10:09:59.411197	\N
2248	150	asset	in_transfer	{}	2026-06-22 10:09:59.412275	\N
2249	157	asset	in_transfer	{}	2026-06-22 10:09:59.413927	\N
2250	255	transfer	delivered	{}	2026-06-22 10:10:01.061656	\N
2251	144	asset	delivered	{}	2026-06-22 10:10:01.063487	\N
2252	149	asset	delivered	{}	2026-06-22 10:10:01.06478	\N
2253	150	asset	delivered	{}	2026-06-22 10:10:01.066103	\N
2254	157	asset	delivered	{}	2026-06-22 10:10:01.067369	\N
2255	256	transfer	in_transfer	{}	2026-06-22 10:10:16.941052	\N
2256	142	asset	in_transfer	{}	2026-06-22 10:10:16.942389	\N
2257	256	transfer	delivered	{}	2026-06-22 10:10:18.688306	\N
2258	142	asset	delivered	{}	2026-06-22 10:10:18.705194	\N
2259	257	transfer	in_transfer	{}	2026-06-22 16:08:00.947055	\N
2260	14	stock	in_transfer	{}	2026-06-22 16:08:00.949701	\N
\.


ALTER TABLE public.audit_logs ENABLE TRIGGER ALL;

--
-- Data for Name: item_category; Type: TABLE DATA; Schema: public; Owner: -
--

ALTER TABLE public.item_category DISABLE TRIGGER ALL;

COPY public.item_category (id, item_category, label, pyr_id, category_type) FROM stdin;
1	laptop	Laptop	LAP	asset
2	printer a4	Drukarka A4	PA4	asset
3	printer a3	Drukarka A3	PA3	asset
4	speakers big	Głośniki (Duże)	SPB	asset
5	speakers small	Głośniki (Małe)	SPS	asset
6	tablet	Tablet	TAB	asset
7	projector	Projektor	PRO	asset
8	power cord	Przedłużacz	POW	stock
9	router	Router	ROU	asset
15	tv	TV	TVX	asset
16	tv 70	TV 70	TV7	asset
17	tv 65	TV 65	TV6	asset
18	telefon	Telefon	TEL	asset
19	skaner kodow	Skaner Kodów	SKA	stock
20	mikrofon	Mikrofon	MIC	asset
21	monitor	Monitor	MON	asset
22	kabel hdmi	Kabel HDMI	HDM	stock
23	etykieciara	Etykieciara	ETY	asset
24	przedłuzacz długi	Przedłużacz długi	PRZ	stock
25	kable audio	Kable Audio	KAB	asset
27	skaner	Skaner	SKA1	asset
28	glosniki i mikrofon	Glosniki i mikrofon	GIM	asset
29	kabel ethernet	Kabel Ethernet	ETH	stock
30	przedłuzacz beben	Przedłużacz bęben	PBE	stock
31	powerbank	Powerbank	PWR	asset
32	kamera	Kamera	CAM	asset
34	stand na tablet	Stand na Tablet	STT	stock
35	ekran do projektora	Ekran do projektora	EDP	stock
36	kabel micro usb	Kabel micro USB	USB	stock
37	kabel usb c	Kabel USB C	USB1	stock
33	tv stick	TV Stick	STI	asset
41	przedłuzacz 5x5	Przedłużacz 5x5	PRZ1	stock
40	ekran projekcyjny	Ekran projekcyjny	EKR	asset
42	stojak na tv	Stojak na TV	SNT	stock
43	tv 75	TV 75	T75	asset
44	kabel jack 3.5	Kabel jack 3.5	3.5	stock
45	drukarka canon	Drukarka Canon	DRU	asset
46	placeholder	placeholder	PLA	stock
47	karta sd	Karta SD	KAR	stock
\.


ALTER TABLE public.item_category ENABLE TRIGGER ALL;

--
-- Data for Name: equipment_request_category_mapping; Type: TABLE DATA; Schema: public; Owner: -
--

ALTER TABLE public.equipment_request_category_mapping DISABLE TRIGGER ALL;

COPY public.equipment_request_category_mapping (id, form_item_name, category_id, created_by, created_at, last_used_at, use_count) FROM stdin;
\.


ALTER TABLE public.equipment_request_category_mapping ENABLE TRIGGER ALL;

--
-- Data for Name: locations; Type: TABLE DATA; Schema: public; Owner: -
--

ALTER TABLE public.locations DISABLE TRIGGER ALL;

COPY public.locations (id, name, details, pavilion) FROM stdin;
3	POW	\N	5
4	RedRoom	\N	10
5	Biuro Akredytacji	\N	WTC
6	Biuro Festiwalowe	\N	10
7	Strefa Cosplay	\N	7A
8	Scena Plenerowa	Plac Marka	PA
9	Gamesroom	\N	6B
11	Strefa Dziecięca	\N	PCC
16	RedPoint PCC	\N	PCC
34	Gród Inicjatyw Fantasytycznych	GIF	3
19	Strefa RPG	\N	11
20	Biuro Prasowe	\N	PCC
22	Integracja	\N	3A
24	Wioski Fantastyczne	\N	3
25	Maskarada	\N	PCC
26	Strefa Warsztatowa	\N	PCC
12	Wypożyczalnia	\N	6A
21	RPGralnia	Iglica	11
27	Sala Warsztatowa	Antresola	7
10	Masterclass	Warsztatowa w 7	7
32	Strefa Fanów Star Wars	\N	8A
30	Akredytacja	zbiorcza	\N
2	HQ	Pawilon 10 na dole	10
31	Pyrsklepik	\N	5A
49	Biuro im. Kamilka	Wejście Północne	EST
40	Strefa B2B	\N	5
41	Sala Ziemii	\N	PCC
50	Sala Taneczna Łany	\N	3
51	Oznakowanie	\N	2
52	Strefa Star Wars	\N	8A
53	Punkt Weryfikacji Broni	\N	WTC
54	Akredytacja Północ-Wschów	Brama 9	BR9
55	Gaming	Gaming wystawcy	6C
56	Strefa Filmowo-serialowa	\N	EST
57	Medycy	Korytasz 8-8A	8A
1	Magazyn Techniczny	Pawilon 2	2
17	Gżdaczroom	\N	2
33	Magazyn	\N	2
28	Krewni Pyrkonu	Plac Marka	PM
13	Akredytacja Wschód	Wejście wschodnie	EST
14	Akredytacja Zachód	Wejście zachodnie	WST
15	Akredytacja Północ	Wejście północne	NRT
37	Szatnia Wschód	\N	EST
38	Szatnia Północ	\N	NRT
35	Szatnie Zachód	\N	WST
43	Strefa Autografów i Foto	\N	8A
44	Sala Kameralna	Pawilon 7	7
23	Strefa Mangi i Anime	Parter	7
45	Strefa Komiksowa	\N	5
46	Strefa Fanów Klocków LEGO	\N	8
47	Strefa Puzzlowa	Antresola	3A
42	Szatnie PCC	Szatnie PCC	PCC
48	Aleja Artystów i Kolekcjonerów	Pawilon 5 - Antresola	5
\.


ALTER TABLE public.locations ENABLE TRIGGER ALL;

--
-- Data for Name: equipment_request_quests; Type: TABLE DATA; Schema: public; Owner: -
--

ALTER TABLE public.equipment_request_quests DISABLE TRIGGER ALL;

COPY public.equipment_request_quests (id, quest_key, quest_id, destination_pavilion, destination_location, recipient, delivery_date, pickup_time, budget_owner, status, last_synced_at, created_at, completed_at, location_id, location_resolved) FROM stdin;
26	Pawilon 3|GIF|Ewa Piotrowski|2026-06-18|	quest-aeb9ae30df1e240f	Pawilon 3	GIF	Ewa Piotrowski	2026-06-18	\N	Budżet C8BA	completed	2026-06-13 23:57:38.705504	2026-06-13 23:57:38.705504	2026-06-17 15:45:42.828716	34	t
95	Pawilon 7|Sala Kameralna|Michał Piotrowski|2026-06-19|12:00	quest-aa9de3c18964fbb7	Pawilon 7	Sala Kameralna	Michał Piotrowski	2026-06-19	12:00	Budżet 4650	completed	2026-06-18 10:26:13.654138	2026-06-18 10:26:13.654138	2026-06-19 10:20:37.155725	44	t
27	Pawilon 3|GIF|Adam Wiśniewski|2026-06-18|	quest-6bc0b7ba43cba386	Pawilon 3	GIF	Adam Wiśniewski	2026-06-18	\N	Budżet 7C1D	completed	2026-06-13 23:57:38.711861	2026-06-13 23:57:38.711861	2026-06-17 15:47:16.866926	34	t
2	Iglica|RPGralnia|Natalia Wójcik|2026-06-18|	quest-334ca5c72d8067c0	Iglica	RPGralnia	Natalia Wójcik	2026-06-18	\N	Budżet F09F	completed	2026-06-13 23:57:38.482058	2026-06-13 23:57:38.482058	2026-06-18 11:55:03.718097	21	t
15	Plac Marka|Krewni Pyrkonu|Łukasz Wiśniewski|2026-06-18|	quest-996da99215e74b0c	Plac Marka	Krewni Pyrkonu	Łukasz Wiśniewski	2026-06-18	\N	Budżet B9F8	completed	2026-06-13 23:57:38.59566	2026-06-13 23:57:38.59566	2026-06-18 11:56:19.745854	28	t
5	Pawilon 5|Strefa komiksowa|Kamil Krawczyk|2026-06-18|	quest-76f4c852354f1018	Pawilon 5	Strefa komiksowa	Kamil Krawczyk	2026-06-18	\N	Budżet 09DB	completed	2026-06-13 23:57:38.51051	2026-06-13 23:57:38.51051	2026-06-18 14:18:04.295643	45	t
32	Pawilon 5|Aleja Artystów i Kolekcjonerów - Antresola|Julia Król|2026-06-19|	quest-8d10f413d9d3fbe6	Pawilon 5	Aleja Artystów i Kolekcjonerów - Antresola	Julia Król	2026-06-19	\N	Budżet 09DB	completed	2026-06-13 23:57:38.747445	2026-06-13 23:57:38.747445	2026-06-18 14:18:10.28602	48	t
38	Pawilon 2|Oznakowanie|Kamil Mazur|2026-06-18|	quest-a33bf6882535a57c	Pawilon 2	Oznakowanie	Kamil Mazur	2026-06-18	\N	Budżet DB72	completed	2026-06-13 23:57:38.791303	2026-06-13 23:57:38.791303	2026-06-17 13:58:49.630108	51	t
16	Wejście wschodnie|Akredytacja|Maria Grabowski|2026-06-17|	quest-945bccac0e1f562c	Wejście wschodnie	Akredytacja	Maria Grabowski	2026-06-17	\N	Budżet D1AB	completed	2026-06-13 23:57:38.60367	2026-06-13 23:57:38.60367	2026-06-18 08:01:07.902998	30	t
11	Wejście północne|Akredytacja|Wojciech Dąbrowski|2026-06-18|	quest-82dd100e9a49933f	Wejście północne	Akredytacja	Wojciech Dąbrowski	2026-06-18	\N	Budżet D1AB	completed	2026-06-13 23:57:38.559897	2026-06-13 23:57:38.559897	2026-06-18 08:01:08.668638	30	t
9	PCC|RedPoint PCC|Tomasz Nowakowski|2026-06-18|	quest-d6b576b204cf65ba	PCC	RedPoint PCC	Tomasz Nowakowski	2026-06-18	\N	Budżet D1AB	completed	2026-06-13 23:57:38.5427	2026-06-13 23:57:38.5427	2026-06-18 15:23:19.734868	16	t
10	Pawilon 2|Gżdaczroom|Joanna Lewandowski|2026-06-18|	quest-e2acfdd55cc905b5	Pawilon 2	Gżdaczroom	Joanna Lewandowski	2026-06-18	\N	Budżet D1AB	completed	2026-06-13 23:57:38.551007	2026-06-13 23:57:38.551007	2026-06-18 09:00:00.273088	17	t
31	Pawilon 10||Tomasz Nowakowski|2026-06-16|	quest-dd264c46a8148ea8	Pawilon 10		Tomasz Nowakowski	2026-06-16	\N	Budżet D1AB	completed	2026-06-13 23:57:38.73957	2026-06-13 23:57:38.73957	2026-06-16 14:54:35.499446	6	t
8	Wejście zachodnie|Akredytacja|Maria Grabowski|2026-06-17|	quest-89897f90580d4fd2	Wejście zachodnie	Akredytacja	Maria Grabowski	2026-06-17	\N	Budżet D1AB	completed	2026-06-13 23:57:38.535801	2026-06-13 23:57:38.535801	2026-06-17 11:06:55.085065	14	t
21	Pawilon 10|RedRoom|Tomasz Nowakowski|2026-06-17|	quest-5fcd84b32e964c86	Pawilon 10	RedRoom	Tomasz Nowakowski	2026-06-17	\N	Budżet D1AB	completed	2026-06-13 23:57:38.65471	2026-06-13 23:57:38.65471	2026-06-17 14:19:48.796936	4	t
48	Pawilon 3a|Strefa Puzzlowa - Antresola|Łukasz Kamiński|2026-06-19|	quest-a1c284b19792cf09	Pawilon 3a	Strefa Puzzlowa - Antresola	Łukasz Kamiński	2026-06-19	\N	Budżet 09DB	completed	2026-06-13 23:57:38.87316	2026-06-13 23:57:38.87316	2026-06-18 18:08:30.616326	47	t
67	Pawilon 3|GIF|Adam Wiśniewski|2026-06-18|10:00	quest-173deb2712ce5d96	Pawilon 3	GIF	Adam Wiśniewski	2026-06-18	10:00	Budżet 7C1D	completed	2026-06-14 10:48:49.943599	2026-06-14 10:48:49.943599	2026-06-17 15:45:37.782691	34	t
66	PWK|Scena Plenerowa|Aleksandra Pawłowski|2026-06-18|12:00	quest-ce6f205d2aa09181	PWK	Scena Plenerowa	Aleksandra Pawłowski	2026-06-18	12:00	Budżet 7C1D	completed	2026-06-14 10:48:49.930165	2026-06-14 10:48:49.930165	2026-06-18 11:56:48.05199	8	t
42	Pawilon 3a|Integracja|Joanna Kwiatkowski|2026-06-17|	quest-b2e7db8422bd567a	Pawilon 3a	Integracja	Joanna Kwiatkowski	2026-06-17	\N	Budżet 4650	completed	2026-06-13 23:57:38.825875	2026-06-13 23:57:38.825875	2026-06-18 12:22:44.139167	22	t
41	Iglica|RPGralnia|Agnieszka Grabowski|2026-06-18|	quest-44710e23103e07c9	Iglica	RPGralnia	Agnieszka Grabowski	2026-06-18	\N	Budżet 9FFA	completed	2026-06-13 23:57:38.81791	2026-06-13 23:57:38.81791	2026-06-18 12:25:49.532472	21	t
57	Iglica|RPGralnia|Agnieszka Grabowski|2026-06-19|	quest-40d82d647bfe2be1	Iglica	RPGralnia	Agnieszka Grabowski	2026-06-19	\N	Budżet 9FFA	completed	2026-06-13 23:57:38.95181	2026-06-13 23:57:38.95181	2026-06-18 12:26:07.852794	21	t
69	Pawilon 10|Biuro Festiwalowe|Natalia Krawczyk|2026-06-18|10:00	quest-01c6b04cf4a555af	Pawilon 10	Biuro Festiwalowe	Natalia Krawczyk	2026-06-18	10:00	Budżet 9ABE	completed	2026-06-14 10:48:49.971703	2026-06-14 10:48:49.971703	2026-06-18 12:26:58.970032	6	t
39	Wejście północne|Weryfikacja|Łukasz Nowak|2026-06-18|	quest-56f37912118a5ce8	Wejście północne	Weryfikacja	Łukasz Nowak	2026-06-18	\N	Budżet D1AB	completed	2026-06-13 23:57:38.799428	2026-06-13 23:57:38.799428	2026-06-18 08:00:29.605639	49	t
45	Wejście północne|Kamilek|Piotr Król|2026-06-17|	quest-2abd9ee775a27361	Wejście północne	Kamilek	Piotr Król	2026-06-17	\N	Budżet D1AB	completed	2026-06-13 23:57:38.85129	2026-06-13 23:57:38.85129	2026-06-18 08:00:34.117861	49	t
36	Wejście północne|Akredytacja|Maria Grabowski|2026-06-17|	quest-43591853569cdf1e	Wejście północne	Akredytacja	Maria Grabowski	2026-06-17	\N	Budżet D1AB	completed	2026-06-13 23:57:38.775249	2026-06-13 23:57:38.775249	2026-06-18 08:00:38.309024	30	t
47	Wejście wschodnie|Szatnia|Magdalena Kamiński|2026-06-18|	quest-ba4820493933d314	Wejście wschodnie	Szatnia	Magdalena Kamiński	2026-06-18	\N	Budżet D1AB	completed	2026-06-13 23:57:38.864201	2026-06-13 23:57:38.864201	2026-06-19 09:19:25.24991	37	t
40	Wejście północne|Szatnia|Magdalena Kamiński|2026-06-18|	quest-581eb51d0e9beae2	Wejście północne	Szatnia	Magdalena Kamiński	2026-06-18	\N	Budżet D1AB	completed	2026-06-13 23:57:38.806623	2026-06-13 23:57:38.806623	2026-06-18 08:00:54.143373	38	t
60	PCC|Szatnia|Magdalena Kamiński|2026-06-18|	quest-a096f98deb5fda12	PCC	Szatnia	Magdalena Kamiński	2026-06-18	\N	Budżet D1AB	completed	2026-06-13 23:57:38.974617	2026-06-13 23:57:38.974617	2026-06-19 09:19:39.603541	42	t
50	Pawilon 3|Wioski Fantastyczne|Julia Woźniak|2026-06-18|	quest-565300dc65df1dea	Pawilon 3	Wioski Fantastyczne	Julia Woźniak	2026-06-18	\N	Budżet 34D4	completed	2026-06-13 23:57:38.889091	2026-06-13 23:57:38.889091	2026-06-18 13:31:51.927805	24	t
54	Wejście zachodnie|Szatnia|Magdalena Kamiński|2026-06-18|	quest-74ca9cd8f26284ed	Wejście zachodnie	Szatnia	Magdalena Kamiński	2026-06-18	\N	Budżet D1AB	completed	2026-06-13 23:57:38.92555	2026-06-13 23:57:38.92555	2026-06-19 09:19:55.880159	35	t
71	PCC|Szatnia|Magdalena Kamiński|2026-06-18|17:00	quest-ff1d0cfab1cca422	PCC	Szatnia	Magdalena Kamiński	2026-06-18	17:00	Budżet D1AB	cancelled	2026-06-14 10:48:49.98658	2026-06-14 10:48:49.98658	\N	42	t
63	PCC|Strefa Dziecięca|Joanna Nowak|2026-06-18|16:00	quest-0fe368e15274ca5d	PCC	Strefa Dziecięca	Joanna Nowak	2026-06-18	16:00	Budżet 4650	completed	2026-06-14 10:48:49.909839	2026-06-14 10:48:49.909839	2026-06-18 13:49:21.23927	11	t
62	Wejście zachodnie|Akredytacja|Bartosz Wójcik|2026-06-18|	quest-d0bd40e11ac83abb	Wejście zachodnie	Akredytacja	Bartosz Wójcik	2026-06-18	\N	Budżet D1AB	completed	2026-06-14 10:48:49.902819	2026-06-14 10:48:49.902819	2026-06-18 08:14:36.013199	14	t
70	Pawilon 8a|Strefa Autografów i Foto|Paweł Kamiński|2026-06-18|16:00	quest-3240059b94ea3e00	Pawilon 8a	Strefa Autografów i Foto	Paweł Kamiński	2026-06-18	16:00	Budżet 9FFA	completed	2026-06-14 10:48:49.979591	2026-06-14 10:48:49.979591	2026-06-18 14:44:49.589122	43	t
49	Pawilon 5|POW|Anna Szymański|2026-06-19|	quest-26775d59ad043647	Pawilon 5	POW	Anna Szymański	2026-06-19	\N	Budżet 50C7	completed	2026-06-13 23:57:38.881195	2026-06-13 23:57:38.881195	2026-06-17 13:25:06.206839	3	t
37	Pawilon 8a|Strefa Autografów i Foto|Paweł Kamiński|2026-06-18|	quest-7455e685bc8ab485	Pawilon 8a	Strefa Autografów i Foto	Paweł Kamiński	2026-06-18	\N	Budżet 9FFA	completed	2026-06-13 23:57:38.782519	2026-06-13 23:57:38.782519	2026-06-18 14:45:08.493508	43	t
59	Wejście północne|Akredytacja|Maria Grabowski|2026-06-11|	quest-537322d885cce98c	Wejście północne	Akredytacja	Maria Grabowski	2026-06-11	\N	Budżet D1AB	completed	2026-06-13 23:57:38.967658	2026-06-13 23:57:38.967658	2026-06-17 14:09:36.269591	30	t
65	Plac Marka|Scena Plenerowa|Aleksandra Pawłowski|2026-06-18|12:00	quest-0010dc4164e3b93a	Plac Marka	Scena Plenerowa	Aleksandra Pawłowski	2026-06-18	12:00	Budżet 7C1D	completed	2026-06-14 10:48:49.9236	2026-06-14 10:48:49.9236	2026-06-18 10:03:50.416241	8	t
64	PCC|Biuro Prasowe|Bartosz Kwiatkowski|2026-06-18|17:00	quest-4765cb79955b0dea	PCC	Biuro Prasowe	Bartosz Kwiatkowski	2026-06-18	17:00	Budżet 9ABE	completed	2026-06-14 10:48:49.917292	2026-06-14 10:48:49.917292	2026-06-18 15:23:10.852774	20	t
68	Pawilon 3a|Integracja|Joanna Kwiatkowski|2026-06-18|16:00	quest-cee9764bfa535831	Pawilon 3a	Integracja	Joanna Kwiatkowski	2026-06-18	16:00	Budżet 7C1D	completed	2026-06-14 10:48:49.953762	2026-06-14 10:48:49.953762	2026-06-18 10:25:44.61946	22	t
61	Pawilon 8|Strefa Fanów Klocków LEGO|Jan Krawczyk|2026-06-18|14:00	quest-8ec4c33a58571329	Pawilon 8	Strefa Fanów Klocków LEGO	Jan Krawczyk	2026-06-18	14:00	Budżet 4650	completed	2026-06-14 10:48:49.875453	2026-06-14 10:48:49.875453	2026-06-18 16:24:43.178006	46	t
92	Pawilon 2|Magazyn|Michał Szymański|2026-06-18|12:00	quest-df25a356d916ca48	Pawilon 2	Magazyn	Michał Szymański	2026-06-18	12:00	Budżet A30D	completed	2026-06-14 10:48:50.213685	2026-06-14 10:48:50.213685	2026-06-18 19:17:15.887896	33	t
94	Pawilon 6b|Gamesroom|Michał Piotrowski|2026-06-18|12:00	quest-dcab40819162e2b6	Pawilon 6b	Gamesroom	Michał Piotrowski	2026-06-18	12:00	Budżet 4650	completed	2026-06-14 10:48:50.231653	2026-06-14 10:48:50.231653	2026-06-18 11:57:21.858629	9	t
80	PCC|Biuro Prasowe|Zofia Lewandowski|2026-06-19|12:00	quest-6d05bd76bbbff90b	PCC	Biuro Prasowe	Zofia Lewandowski	2026-06-19	12:00	Budżet 3563	completed	2026-06-14 10:48:50.098954	2026-06-14 10:48:50.098954	2026-06-18 19:18:05.488631	20	t
73	Pawilon 7|Strefa Mangi i Anime|Zofia Lewandowski|2026-06-19|10:00	quest-fdb79a8ccd294c07	Pawilon 7	Strefa Mangi i Anime	Zofia Lewandowski	2026-06-19	10:00	Budżet 7C1D	completed	2026-06-14 10:48:50.012719	2026-06-14 10:48:50.012719	2026-06-19 06:42:54.602127	23	t
90	Pawilon 10|Biuro Festiwalowe|Paweł Jankowski|2026-06-18|10:00	quest-d1ff2d073d0946a5	Pawilon 10	Biuro Festiwalowe	Paweł Jankowski	2026-06-18	10:00	Budżet 9ABE	completed	2026-06-14 10:48:50.192725	2026-06-14 10:48:50.192725	2026-06-18 13:40:30.767424	6	t
78	PCC|Sala Warsztatowa|Katarzyna Król|2026-06-18|16:00	quest-ea3ddb20b8134840	PCC	Sala Warsztatowa	Katarzyna Król	2026-06-18	16:00	Budżet 9FFA	completed	2026-06-14 10:48:50.08216	2026-06-14 10:48:50.08216	2026-06-19 13:09:47.87091	26	t
88	PCC|Maskarada|Ewa Mazur|2026-06-19|10:00	quest-00ae0dc918b16c6f	PCC	Maskarada	Ewa Mazur	2026-06-19	10:00	Budżet 2595	completed	2026-06-14 10:48:50.172732	2026-06-14 10:48:50.172732	2026-06-19 14:59:12.272565	25	t
82	Pawilon 2|Magazyn Techniczny|Zofia Zieliński|2026-06-17|16:00	quest-c35ab70d67ad7688	Pawilon 2	Magazyn Techniczny	Zofia Zieliński	2026-06-17	16:00	Budżet D1AB	cancelled	2026-06-14 10:48:50.115688	2026-06-14 10:48:50.115688	\N	1	t
77	Pawilon 5|Strefa komiksowa|Joanna Nowak|2026-06-17|10:00	quest-422eae9ff39e977c	Pawilon 5	Strefa komiksowa	Joanna Nowak	2026-06-17	10:00	Budżet 4650	completed	2026-06-14 10:48:50.069421	2026-06-14 10:48:50.069421	2026-06-17 16:06:07.156302	45	t
83	Pawilon 5|Strefa komiksowa|Kamil Krawczyk|2026-06-19|12:00	quest-d77d641aadf22dcf	Pawilon 5	Strefa komiksowa	Kamil Krawczyk	2026-06-19	12:00	Budżet 09DB	completed	2026-06-14 10:48:50.121447	2026-06-14 10:48:50.121447	2026-06-18 14:17:59.910359	45	t
86	Pawilon 5|POW|Kamil Krawczyk|2026-06-16|09:00	quest-f1f5fa761ea39cb5	Pawilon 5	POW	Kamil Krawczyk	2026-06-16	09:00	Budżet 50C7	completed	2026-06-14 10:48:50.151506	2026-06-14 10:48:50.151506	2026-06-17 13:24:33.419162	3	t
87	Pawilon 7|Strefa Mangi i Anime|Magdalena Woźniak|2026-06-18|16:00	quest-05d113267d475d32	Pawilon 7	Strefa Mangi i Anime	Magdalena Woźniak	2026-06-18	16:00	Budżet 4650	completed	2026-06-14 10:48:50.166293	2026-06-14 10:48:50.166293	2026-06-18 14:24:12.760354	23	t
93	Pawilon 5|POW|Kamil Krawczyk|2026-06-17|12:00	quest-7a7217652b90e45f	Pawilon 5	POW	Kamil Krawczyk	2026-06-17	12:00	Budżet 50C7	completed	2026-06-14 10:48:50.223442	2026-06-14 10:48:50.223442	2026-06-17 13:24:54.755709	3	t
89	Pawilon 7|Sala Warsztatowa|Marcin Jankowski|2026-06-18|16:00	quest-f424c0a43ac11537	Pawilon 7	Sala Warsztatowa	Marcin Jankowski	2026-06-18	16:00	Budżet 9FFA	completed	2026-06-14 10:48:50.185186	2026-06-14 10:48:50.185186	2026-06-18 14:24:35.495862	27	t
75	Pawilon 7a|Strefa Cosplay|Ewa Mazur|2026-06-18|14:00	quest-6ad1d4815d784c36	Pawilon 7a	Strefa Cosplay	Ewa Mazur	2026-06-18	14:00	Budżet 2595	completed	2026-06-14 10:48:50.039408	2026-06-14 10:48:50.039408	2026-06-17 19:17:25.411705	7	t
79	Pawilon 8|Strefa Fanów Klocków LEGO|Jan Krawczyk|2026-06-18|12:00	quest-2a88469f39ea8692	Pawilon 8	Strefa Fanów Klocków LEGO	Jan Krawczyk	2026-06-18	12:00	Budżet 4650	completed	2026-06-14 10:48:50.089688	2026-06-14 10:48:50.089688	2026-06-18 14:44:08.747162	46	t
81	Pawilon 8|Strefa Fanów Klocków LEGO|Jan Krawczyk|2026-06-18|17:00	quest-871913dafd5ec84e	Pawilon 8	Strefa Fanów Klocków LEGO	Jan Krawczyk	2026-06-18	17:00	Budżet 4650	completed	2026-06-14 10:48:50.108264	2026-06-14 10:48:50.108264	2026-06-18 14:44:27.095102	46	t
84	Wejście północne|Akredytacja|Bartosz Wójcik|2026-06-18|	quest-db39299c4c9bcef0	Wejście północne	Akredytacja	Bartosz Wójcik	2026-06-18	\N	Budżet D1AB	completed	2026-06-14 10:48:50.135315	2026-06-14 10:48:50.135315	2026-06-18 08:00:45.862462	30	t
91	Wejście wschodnie|Akredytacja|Bartosz Wójcik|2026-06-18|	quest-69085d87018356ed	Wejście wschodnie	Akredytacja	Bartosz Wójcik	2026-06-18	\N	Budżet D1AB	completed	2026-06-14 10:48:50.20514	2026-06-14 10:48:50.20514	2026-06-18 08:01:16.639559	30	t
85	Pawilon 10|HQ|Adam Zieliński|2026-06-17|16:00	quest-458151fb13775fb8	Pawilon 10	HQ	Adam Zieliński	2026-06-17	16:00	Budżet A30D	completed	2026-06-14 10:48:50.143017	2026-06-14 10:48:50.143017	2026-06-18 08:39:37.54639	2	t
74	Pawilon 6a|Wypożyczalnia|Aleksandra Jankowski|2026-06-18|10:00	quest-2be911325aa96e09	Pawilon 6a	Wypożyczalnia	Aleksandra Jankowski	2026-06-18	10:00	Budżet 4650	completed	2026-06-14 10:48:50.024588	2026-06-14 10:48:50.024588	2026-06-18 08:44:55.121838	12	t
72	Pawilon 3|Antresola|Agnieszka Szymański|2026-06-18|15:00	quest-f003bc89b0fab413	Pawilon 3	Antresola	Agnieszka Szymański	2026-06-18	15:00	Budżet 7C1D	completed	2026-06-14 10:48:49.996506	2026-06-14 10:48:49.996506	2026-06-18 10:03:10.421764	22	t
\.


ALTER TABLE public.equipment_request_quests ENABLE TRIGGER ALL;

--
-- Data for Name: equipment_request_items; Type: TABLE DATA; Schema: public; Owner: -
--

ALTER TABLE public.equipment_request_items DISABLE TRIGGER ALL;

COPY public.equipment_request_items (id, quest_id, item_name, quantity, category_id, category_match_type, category_match_confidence, budget_owner, notes, source_row_number, created_at) FROM stdin;
250	95	Głośniki (Duże)	1	4	exact	\N	\N	\N	43	2026-06-18 10:26:13.654138
2	2	Drukarka A4	2	2	exact	\N	\N	\N	82	2026-06-13 23:57:38.482058
3	2	Laptop	4	1	exact	\N	\N	\N	83	2026-06-13 23:57:38.482058
4	2	Przedłużacz	15	8	exact	\N	\N	\N	84	2026-06-13 23:57:38.482058
5	2	Tablet	3	6	exact	\N	\N	\N	85	2026-06-13 23:57:38.482058
6	2	Telefon	1	18	exact	\N	\N	\N	86	2026-06-13 23:57:38.482058
7	2	Router	1	9	exact	\N	\N	\N	154	2026-06-13 23:57:38.482058
251	95	Laptop	1	1	exact	\N	\N	Uwagi do pozycji 251	44	2026-06-18 10:26:13.654138
252	95	Mikrofon	1	20	exact	\N	\N	\N	45	2026-06-18 10:26:13.654138
253	95	Przedłużacz	4	8	exact	\N	\N	\N	46	2026-06-18 10:26:13.654138
254	95	TV 55'	1	17	fuzzy	\N	\N	Uwagi do pozycji 254	47	2026-06-18 10:26:13.654138
12	5	Przedłużacz	3	8	exact	\N	\N	Uwagi do pozycji 12	114	2026-06-13 23:57:38.51051
13	5	Telefon	1	18	exact	\N	\N	Uwagi do pozycji 13	115	2026-06-13 23:57:38.51051
20	8	Drukarka A4	2	2	exact	\N	\N	\N	80	2026-06-13 23:57:38.535801
21	9	Przedłużacz	3	8	exact	\N	\N	\N	92	2026-06-13 23:57:38.5427
22	9	Drukarka A4	1	2	exact	\N	\N	\N	93	2026-06-13 23:57:38.5427
23	9	Laptop	1	1	exact	\N	\N	\N	94	2026-06-13 23:57:38.5427
24	10	Drukarka A4	1	2	exact	\N	\N	Uwagi do pozycji 24	102	2026-06-13 23:57:38.551007
25	10	Cartridge do drukarki	1	\N	none	\N	\N	Uwagi do pozycji 25	103	2026-06-13 23:57:38.551007
26	10	Głośniki (male)	1	5	fuzzy	\N	\N	\N	104	2026-06-13 23:57:38.551007
27	10	Laptop	4	1	exact	\N	\N	Uwagi do pozycji 27	105	2026-06-13 23:57:38.551007
28	10	Przedłużacz	15	8	exact	\N	\N	Uwagi do pozycji 28	106	2026-06-13 23:57:38.551007
29	10	Telefon	3	18	exact	\N	\N	Uwagi do pozycji 29	107	2026-06-13 23:57:38.551007
30	10	TV 55'	1	17	fuzzy	\N	\N	Uwagi do pozycji 30	108	2026-06-13 23:57:38.551007
31	11	Telefon	4	18	exact	\N	\N	Uwagi do pozycji 31	113	2026-06-13 23:57:38.559897
36	15	Przedłużacz	5	8	exact	\N	\N	Uwagi do pozycji 36	112	2026-06-13 23:57:38.59566
37	16	Router	1	9	exact	\N	\N	\N	77	2026-06-13 23:57:38.60367
38	16	Przedłużacz	4	8	exact	\N	\N	\N	78	2026-06-13 23:57:38.60367
59	21	Drukarka A3	1	3	exact	\N	\N	Uwagi do pozycji 59	88	2026-06-13 23:57:38.65471
60	21	Drukarka A4	1	2	exact	\N	\N	Uwagi do pozycji 60	89	2026-06-13 23:57:38.65471
61	21	Laptop	2	1	exact	\N	\N	\N	90	2026-06-13 23:57:38.65471
62	21	Przedłużacz	5	8	exact	\N	\N	Uwagi do pozycji 62	91	2026-06-13 23:57:38.65471
75	26	Drukarka A4	1	2	exact	\N	\N	\N	150	2026-06-13 23:57:38.705504
76	26	Głośniki (Duże)	1	4	exact	\N	\N	Uwagi do pozycji 76	151	2026-06-13 23:57:38.705504
77	26	Laptop	2	1	exact	\N	\N	\N	152	2026-06-13 23:57:38.705504
78	26	Przedłużacz	6	8	exact	\N	\N	\N	153	2026-06-13 23:57:38.705504
79	27	TV 70'	1	16	fuzzy	\N	\N	Uwagi do pozycji 79	159	2026-06-13 23:57:38.711861
80	27	TV 45'	1	17	fuzzy	\N	\N	\N	160	2026-06-13 23:57:38.711861
81	27	TV 55'	4	17	fuzzy	\N	\N	\N	161	2026-06-13 23:57:38.711861
85	31	Projektor z Ekranem	1	\N	none	\N	\N	Uwagi do pozycji 85	95	2026-06-13 23:57:38.73957
86	32	Przedłużacz	10	8	exact	\N	\N	\N	130	2026-06-13 23:57:38.747445
96	36	Router	1	9	exact	\N	\N	\N	76	2026-06-13 23:57:38.775249
97	36	Telefon	1	18	exact	\N	\N	\N	79	2026-06-13 23:57:38.775249
98	37	Router	1	9	exact	\N	\N	\N	116	2026-06-13 23:57:38.782519
99	38	Przedłużacz	1	8	exact	\N	\N	\N	109	2026-06-13 23:57:38.791303
100	38	Drukarka A3	1	3	exact	\N	\N	Uwagi do pozycji 100	110	2026-06-13 23:57:38.791303
101	39	Przedłużacz	3	8	exact	\N	\N	\N	145	2026-06-13 23:57:38.799428
102	39	Drukarka A4	1	2	exact	\N	\N	Uwagi do pozycji 102	146	2026-06-13 23:57:38.799428
103	40	Przedłużacz	1	8	exact	\N	\N	Uwagi do pozycji 103	120	2026-06-13 23:57:38.806623
104	40	Przedłużacz	1	8	exact	\N	\N	Uwagi do pozycji 104	124	2026-06-13 23:57:38.806623
105	41	Laptop	3	1	exact	\N	\N	Uwagi do pozycji 105	131	2026-06-13 23:57:38.81791
106	41	Przedłużacz	2	8	exact	\N	\N	Uwagi do pozycji 106	132	2026-06-13 23:57:38.81791
107	41	Głośniki (Duże)	1	4	exact	\N	\N	Uwagi do pozycji 107	133	2026-06-13 23:57:38.81791
108	42	TV 70'	1	16	fuzzy	\N	\N	Uwagi do pozycji 108	70	2026-06-13 23:57:38.825875
109	42	Laptop	1	1	exact	\N	\N	Uwagi do pozycji 109	71	2026-06-13 23:57:38.825875
110	42	Przedłużacz	40	8	exact	\N	\N	\N	72	2026-06-13 23:57:38.825875
111	42	Kabel jack 3.5 mm	1	\N	none	\N	\N	\N	73	2026-06-13 23:57:38.825875
112	42	Cartridge do drukarki	3	\N	none	\N	\N	Uwagi do pozycji 112	74	2026-06-13 23:57:38.825875
120	45	Monitor	1	21	exact	\N	\N	\N	81	2026-06-13 23:57:38.85129
122	47	Przedłużacz	2	8	exact	\N	\N	Uwagi do pozycji 122	117	2026-06-13 23:57:38.864201
123	47	Laptop	1	1	exact	\N	\N	Uwagi do pozycji 123	122	2026-06-13 23:57:38.864201
124	48	Przedłużacz	2	8	exact	\N	\N	Uwagi do pozycji 124	125	2026-06-13 23:57:38.87316
125	48	Router	1	9	exact	\N	\N	Uwagi do pozycji 125	126	2026-06-13 23:57:38.87316
126	49	Przedłużacz	5	8	exact	\N	\N	\N	129	2026-06-13 23:57:38.881195
127	50	Przedłużacz	30	8	exact	\N	\N	Uwagi do pozycji 127	147	2026-06-13 23:57:38.889091
128	50	TV 55'	3	17	fuzzy	\N	\N	Uwagi do pozycji 128	148	2026-06-13 23:57:38.889091
129	50	Laptop	1	1	exact	\N	\N	Uwagi do pozycji 129	149	2026-06-13 23:57:38.889091
141	54	Przedłużacz	2	8	exact	\N	\N	Uwagi do pozycji 141	118	2026-06-13 23:57:38.92555
142	54	Laptop	2	1	exact	\N	\N	Uwagi do pozycji 142	121	2026-06-13 23:57:38.92555
153	57	Projektor / TV	1	\N	none	\N	\N	Uwagi do pozycji 153	134	2026-06-13 23:57:38.95181
155	59	Laptop	11	1	exact	\N	\N	\N	75	2026-06-13 23:57:38.967658
156	60	Przedłużacz	2	8	exact	\N	\N	Uwagi do pozycji 156	119	2026-06-13 23:57:38.974617
157	60	Laptop	1	1	exact	\N	\N	Uwagi do pozycji 157	123	2026-06-13 23:57:38.974617
158	61	TV 55'	2	17	fuzzy	\N	\N	Uwagi do pozycji 158	96	2026-06-14 10:48:49.875453
159	61	TV 45'	2	17	fuzzy	\N	\N	Uwagi do pozycji 159	97	2026-06-14 10:48:49.875453
160	61	Monitor	3	21	exact	\N	\N	\N	100	2026-06-14 10:48:49.875453
161	62	Laptop	\N	1	exact	\N	\N	Uwagi do pozycji 161	136	2026-06-14 10:48:49.902819
162	63	Laptop	5	1	exact	\N	\N	Uwagi do pozycji 162	6	2026-06-14 10:48:49.909839
163	63	Drukarka A4	1	2	exact	\N	\N	\N	7	2026-06-14 10:48:49.909839
164	63	Przedłużacz	4	8	exact	\N	\N	\N	8	2026-06-14 10:48:49.909839
165	63	Głośniki (male)	1	5	fuzzy	\N	\N	\N	9	2026-06-14 10:48:49.909839
166	63	Telefon	1	18	exact	\N	\N	\N	10	2026-06-14 10:48:49.909839
167	63	TV 55'	1	17	fuzzy	\N	\N	Uwagi do pozycji 167	11	2026-06-14 10:48:49.909839
168	64	Drukarka A4	1	2	exact	\N	\N	\N	29	2026-06-14 10:48:49.917292
169	64	Przedłużacz	30	8	exact	\N	\N	Uwagi do pozycji 169	30	2026-06-14 10:48:49.917292
170	64	Router	2	9	exact	\N	\N	\N	31	2026-06-14 10:48:49.917292
171	65	Przedłużacz	18	8	exact	\N	\N	Uwagi do pozycji 171	32	2026-06-14 10:48:49.9236
172	66	Przedłużacz	10	8	exact	\N	\N	Uwagi do pozycji 172	33	2026-06-14 10:48:49.930165
173	67	Router	1	9	exact	\N	\N	\N	155	2026-06-14 10:48:49.943599
174	67	Przedłużacz	2	8	exact	\N	\N	Uwagi do pozycji 174	156	2026-06-14 10:48:49.943599
175	67	Przedłużacz	23	8	exact	\N	\N	Uwagi do pozycji 175	157	2026-06-14 10:48:49.943599
176	67	Przedłużacz	2	8	exact	\N	\N	Uwagi do pozycji 176	158	2026-06-14 10:48:49.943599
177	68	TV 45'	1	17	fuzzy	\N	\N	Uwagi do pozycji 177	34	2026-06-14 10:48:49.953762
178	68	TV 55'	2	17	fuzzy	\N	\N	\N	35	2026-06-14 10:48:49.953762
179	69	inne	3	\N	none	\N	\N	Uwagi do pozycji 179	127	2026-06-14 10:48:49.971703
180	69	Telefon	1	18	exact	\N	\N	Uwagi do pozycji 180	128	2026-06-14 10:48:49.971703
181	70	Przedłużacz	15	8	exact	\N	\N	\N	57	2026-06-14 10:48:49.979591
182	70	Laptop	1	1	exact	\N	\N	\N	87	2026-06-14 10:48:49.979591
183	71	Telefon	1	18	exact	\N	\N	Uwagi do pozycji 183	101	2026-06-14 10:48:49.98658
184	72	Monitor	2	21	exact	\N	\N	Uwagi do pozycji 184	111	2026-06-14 10:48:49.996506
185	73	Przedłużacz	50	8	exact	\N	\N	\N	138	2026-06-14 10:48:50.012719
186	73	TV 55'	6	17	fuzzy	\N	\N	\N	139	2026-06-14 10:48:50.012719
187	73	Monitor	13	21	exact	\N	\N	\N	140	2026-06-14 10:48:50.012719
188	73	Projektor z Ekranem	1	\N	none	\N	\N	\N	141	2026-06-14 10:48:50.012719
189	73	Głośniki (male)	1	5	fuzzy	\N	\N	\N	142	2026-06-14 10:48:50.012719
190	73	TV 45'	1	17	fuzzy	\N	\N	\N	143	2026-06-14 10:48:50.012719
191	73	Laptop	1	1	exact	\N	\N	Uwagi do pozycji 191	144	2026-06-14 10:48:50.012719
192	74	Laptop	10	1	exact	\N	\N	\N	58	2026-06-14 10:48:50.024588
193	74	Przedłużacz	15	8	exact	\N	\N	\N	59	2026-06-14 10:48:50.024588
194	74	Router	1	9	exact	\N	\N	\N	60	2026-06-14 10:48:50.024588
195	74	Skaner Kodów	12	19	exact	\N	\N	\N	61	2026-06-14 10:48:50.024588
196	75	Drukarka A4	1	2	exact	\N	\N	Uwagi do pozycji 196	50	2026-06-14 10:48:50.039408
197	75	Przedłużacz	30	8	exact	\N	\N	Uwagi do pozycji 197	51	2026-06-14 10:48:50.039408
203	77	Router	1	9	exact	\N	\N	\N	62	2026-06-14 10:48:50.069421
204	78	Laptop	6	1	exact	\N	\N	Uwagi do pozycji 204	17	2026-06-14 10:48:50.08216
205	78	Drukarka A4	1	2	exact	\N	\N	\N	19	2026-06-14 10:48:50.08216
206	78	Przedłużacz	10	8	exact	\N	\N	Uwagi do pozycji 206	20	2026-06-14 10:48:50.08216
207	78	TV 55'	6	17	fuzzy	\N	\N	\N	22	2026-06-14 10:48:50.08216
208	79	Przedłużacz	60	8	exact	\N	\N	\N	98	2026-06-14 10:48:50.089688
209	80	Tablet	8	6	exact	\N	\N	Uwagi do pozycji 209	27	2026-06-14 10:48:50.098954
210	81	Telefon	2	18	exact	\N	\N	\N	99	2026-06-14 10:48:50.108264
211	82	Monitor	4	21	exact	\N	\N	Uwagi do pozycji 211	2	2026-06-14 10:48:50.115688
212	82	TV 70'	1	16	fuzzy	\N	\N	\N	3	2026-06-14 10:48:50.115688
213	82	Drukarka A4	1	2	exact	\N	\N	\N	4	2026-06-14 10:48:50.115688
214	82	Router	1	9	exact	\N	\N	Uwagi do pozycji 214	5	2026-06-14 10:48:50.115688
215	83	Laptop	1	1	exact	\N	\N	Uwagi do pozycji 215	24	2026-06-14 10:48:50.121447
216	84	Laptop	\N	1	exact	\N	\N	Uwagi do pozycji 216	137	2026-06-14 10:48:50.135315
217	85	Drukarka A4	1	2	exact	\N	\N	Uwagi do pozycji 217	12	2026-06-14 10:48:50.143017
218	85	Laptop	3	1	exact	\N	\N	\N	13	2026-06-14 10:48:50.143017
219	85	Przedłużacz	10	8	exact	\N	\N	Uwagi do pozycji 219	14	2026-06-14 10:48:50.143017
220	85	Tablet	2	6	exact	\N	\N	Uwagi do pozycji 220	15	2026-06-14 10:48:50.143017
221	85	Telefon	5	18	exact	\N	\N	Uwagi do pozycji 221	16	2026-06-14 10:48:50.143017
222	86	Router	1	9	exact	\N	\N	\N	26	2026-06-14 10:48:50.151506
223	87	Przedłużacz	1	8	exact	\N	\N	Uwagi do pozycji 223	28	2026-06-14 10:48:50.166293
224	88	Laptop	3	1	exact	\N	\N	Uwagi do pozycji 224	49	2026-06-14 10:48:50.172732
225	89	Laptop	2	1	exact	\N	\N	Uwagi do pozycji 225	18	2026-06-14 10:48:50.185186
226	89	Przedłużacz	2	8	exact	\N	\N	Uwagi do pozycji 226	21	2026-06-14 10:48:50.185186
227	89	TV 55'	2	17	fuzzy	\N	\N	\N	23	2026-06-14 10:48:50.185186
228	90	Laptop	11	1	exact	\N	\N	Uwagi do pozycji 228	63	2026-06-14 10:48:50.192725
229	90	Drukarka A4	2	2	exact	\N	\N	Uwagi do pozycji 229	64	2026-06-14 10:48:50.192725
230	90	Router	1	9	exact	\N	\N	Uwagi do pozycji 230	65	2026-06-14 10:48:50.192725
231	90	Tablet	3	6	exact	\N	\N	Uwagi do pozycji 231	66	2026-06-14 10:48:50.192725
232	90	Przedłużacz	20	8	exact	\N	\N	Uwagi do pozycji 232	67	2026-06-14 10:48:50.192725
233	90	Przedłużacz	6	8	exact	\N	\N	Uwagi do pozycji 233	68	2026-06-14 10:48:50.192725
234	90	Telefon	6	18	exact	\N	\N	Uwagi do pozycji 234	69	2026-06-14 10:48:50.192725
235	91	Laptop	\N	1	exact	\N	\N	Uwagi do pozycji 235	135	2026-06-14 10:48:50.20514
236	92	Laptop	2	1	exact	\N	\N	\N	52	2026-06-14 10:48:50.213685
237	92	Drukarka A4	1	2	exact	\N	\N	\N	53	2026-06-14 10:48:50.213685
238	92	Telefon	1	18	exact	\N	\N	\N	54	2026-06-14 10:48:50.213685
239	92	TV 70'	1	16	fuzzy	\N	\N	Uwagi do pozycji 239	55	2026-06-14 10:48:50.213685
240	92	Przedłużacz	3	8	exact	\N	\N	\N	56	2026-06-14 10:48:50.213685
241	93	Drukarka A4	1	2	exact	\N	\N	\N	25	2026-06-14 10:48:50.223442
242	94	Drukarka A4	2	2	exact	\N	\N	Uwagi do pozycji 242	36	2026-06-14 10:48:50.231653
243	94	Głośniki (Duże)	2	4	exact	\N	\N	\N	37	2026-06-14 10:48:50.231653
244	94	Laptop	4	1	exact	\N	\N	Uwagi do pozycji 244	38	2026-06-14 10:48:50.231653
245	94	Mikrofon	2	20	exact	\N	\N	\N	39	2026-06-14 10:48:50.231653
246	94	Przedłużacz	40	8	exact	\N	\N	\N	40	2026-06-14 10:48:50.231653
247	94	TV 55'	7	17	fuzzy	\N	\N	Uwagi do pozycji 247	41	2026-06-14 10:48:50.231653
248	94	TV 70'	1	16	fuzzy	\N	\N	Uwagi do pozycji 248	42	2026-06-14 10:48:50.231653
249	94	kamerka	1	32	fuzzy	\N	\N	Uwagi do pozycji 249	48	2026-06-14 10:48:50.231653
\.


ALTER TABLE public.equipment_request_items ENABLE TRIGGER ALL;

--
-- Data for Name: equipment_request_location_mapping; Type: TABLE DATA; Schema: public; Owner: -
--

ALTER TABLE public.equipment_request_location_mapping DISABLE TRIGGER ALL;

COPY public.equipment_request_location_mapping (id, pavilion, location_name, location_id, created_at, usage_count) FROM stdin;
1	Wejście zachodnie	Akredytacja	14	2026-02-22 19:40:36.802779	25
14	Pawilon 3	GIF	34	2026-05-26 08:28:40.101905	5
6	Pawilon 8a	Strefa Autografów i Foto	43	2026-05-12 19:26:42.888607	6
5	PCC	Szatnia	42	2026-03-05 22:27:27.465525	13
9	Pawilon 5	Strefa komiksowa	45	2026-05-26 08:26:58.441106	622
18	Pawilon 5	Aleja Artystów i Kolekcjonerów - Antresola	48	2026-06-14 21:16:55.980557	0
20	Wejście północne	Kamilek	49	2026-06-14 22:01:11.47906	0
21	Pawilon 2	Oznakowanie	51	2026-06-17 10:12:08.225874	0
22	Pawilon 3	Antresola	22	2026-06-18 09:15:47.422072	0
15	Pawilon 7	Sala Kameralna	44	2026-05-26 08:29:07.343598	3
23	PCC	Sala Warsztatowa	26	2026-06-19 13:11:28.79802	0
16	Pawilon 10		6	2026-05-26 08:29:30.889894	0
2	Wejście północne	Szatnia	38	2026-02-22 23:09:54.568452	965
3	Wejście wschodnie	Szatnia	37	2026-02-24 23:23:39.860426	7
13	Pawilon 3a	Strefa Puzzlowa - Antresola	47	2026-05-26 08:28:11.514436	1
10	Wejście zachodnie	Szatnia	35	2026-05-26 08:27:16.006367	1
\.


ALTER TABLE public.equipment_request_location_mapping ENABLE TRIGGER ALL;

--
-- Data for Name: equipment_request_price_list; Type: TABLE DATA; Schema: public; Owner: -
--

ALTER TABLE public.equipment_request_price_list DISABLE TRIGGER ALL;

COPY public.equipment_request_price_list (id, item_name, supplier, unit_price, updated_at) FROM stdin;
1	Monitor	Oki Event	70.00	2026-08-06 21:30:47.614613
2	TV 70'	Probis	813.00	2026-08-06 21:30:47.616715
3	TV 70'	Netland	813.00	2026-08-06 21:30:47.618373
4	TV 70'	Oki Event	1400.00	2026-08-06 21:30:47.619898
5	Drukarka A4	Probis	160.00	2026-08-06 21:30:47.621614
6	Laptop	Probis	120.00	2026-08-06 21:30:47.623246
7	Laptop	Netland	90.00	2026-08-06 21:30:47.624896
8	TV 55'	Probis	500.00	2026-08-06 21:30:47.626335
9	TV 55'	Netland	300.00	2026-08-06 21:30:47.628813
10	TV 55'	Oki Event	1200.00	2026-08-06 21:30:47.630626
11	TV 45'	Oki Event	800.00	2026-08-06 21:30:47.632509
12	Głośniki (Duże)	Oki Event	300.00	2026-08-06 21:30:47.634293
14	Projektor / TV	Probis	300.00	2026-08-06 21:30:47.635709
28	Projektor z Ekranem	Probis	200.00	2026-06-18 10:33:49.347998
13	Drukarka A3	Probis	200.00	2026-08-06 21:30:47.637527
\.


ALTER TABLE public.equipment_request_price_list ENABLE TRIGGER ALL;

--
-- Data for Name: equipment_request_sync_log; Type: TABLE DATA; Schema: public; Owner: -
--

ALTER TABLE public.equipment_request_sync_log DISABLE TRIGGER ALL;

COPY public.equipment_request_sync_log (id, synced_at, rows_processed, quests_created, quests_updated, quests_unchanged, items_added, items_removed, errors, success, duration_ms, sheet_id) FROM stdin;
7361	2026-08-06 09:15:46.836746	160	0	0	63	0	0	\N	t	1514	fixture-sheet-id
7362	2026-08-06 09:30:46.840392	160	0	0	63	0	0	\N	t	1519	fixture-sheet-id
7363	2026-08-06 09:45:46.663911	160	0	0	63	0	0	\N	t	1343	fixture-sheet-id
7364	2026-08-06 10:00:46.706369	160	0	0	63	0	0	\N	t	1386	fixture-sheet-id
7365	2026-08-06 10:15:46.85058	160	0	0	63	0	0	\N	t	1530	fixture-sheet-id
7366	2026-08-06 10:30:46.738626	160	0	0	63	0	0	\N	t	1418	fixture-sheet-id
7367	2026-08-06 10:45:46.720614	160	0	0	63	0	0	\N	t	1400	fixture-sheet-id
7368	2026-08-06 11:00:46.837929	160	0	0	63	0	0	\N	t	1517	fixture-sheet-id
7369	2026-08-06 11:15:46.657914	160	0	0	63	0	0	\N	t	1336	fixture-sheet-id
7370	2026-08-06 11:30:47.083803	160	0	0	63	0	0	\N	t	1760	fixture-sheet-id
7371	2026-08-06 11:45:46.765811	160	0	0	63	0	0	\N	t	1444	fixture-sheet-id
7372	2026-08-06 12:00:46.971786	160	0	0	63	0	0	\N	t	1648	fixture-sheet-id
7373	2026-08-06 12:15:48.154049	160	0	0	63	0	0	\N	t	2832	fixture-sheet-id
7374	2026-08-06 12:30:47.512965	160	0	0	63	0	0	\N	t	2190	fixture-sheet-id
7375	2026-08-06 12:45:46.836918	160	0	0	63	0	0	\N	t	1514	fixture-sheet-id
7376	2026-08-06 13:00:47.321311	160	0	0	63	0	0	\N	t	1995	fixture-sheet-id
7377	2026-08-06 13:15:46.820946	160	0	0	63	0	0	\N	t	1498	fixture-sheet-id
7378	2026-08-06 13:30:47.688774	160	0	0	63	0	0	\N	t	2365	fixture-sheet-id
7379	2026-08-06 13:45:53.798766	160	0	0	63	0	0	\N	t	8477	fixture-sheet-id
7380	2026-08-06 14:00:46.974865	160	0	0	63	0	0	\N	t	1652	fixture-sheet-id
7381	2026-08-06 14:15:46.760945	160	0	0	63	0	0	\N	t	1440	fixture-sheet-id
7382	2026-08-06 14:30:47.458835	160	0	0	63	0	0	\N	t	2138	fixture-sheet-id
7383	2026-08-06 14:45:46.661952	160	0	0	63	0	0	\N	t	1341	fixture-sheet-id
7384	2026-08-06 15:00:47.768841	160	0	0	63	0	0	\N	t	2447	fixture-sheet-id
7385	2026-08-06 15:15:46.999724	160	0	0	63	0	0	\N	t	1678	fixture-sheet-id
7386	2026-08-06 15:30:46.944988	160	0	0	63	0	0	\N	t	1624	fixture-sheet-id
7387	2026-08-06 15:45:46.646346	160	0	0	63	0	0	\N	t	1321	fixture-sheet-id
7388	2026-08-06 16:00:47.386111	160	0	0	63	0	0	\N	t	2065	fixture-sheet-id
7389	2026-08-06 16:15:46.661756	160	0	0	63	0	0	\N	t	1340	fixture-sheet-id
7390	2026-08-06 16:30:47.413313	160	0	0	63	0	0	\N	t	2092	fixture-sheet-id
7391	2026-08-06 16:45:47.01661	160	0	0	63	0	0	\N	t	1695	fixture-sheet-id
7392	2026-08-06 17:00:46.923691	160	0	0	63	0	0	\N	t	1603	fixture-sheet-id
7393	2026-08-06 17:15:46.710095	160	0	0	63	0	0	\N	t	1390	fixture-sheet-id
7394	2026-08-06 17:30:46.9701	160	0	0	63	0	0	\N	t	1648	fixture-sheet-id
7395	2026-08-06 17:45:46.71166	160	0	0	63	0	0	\N	t	1391	fixture-sheet-id
7396	2026-08-06 18:00:46.775954	160	0	0	63	0	0	\N	t	1455	fixture-sheet-id
7397	2026-08-06 18:15:46.717684	160	0	0	63	0	0	\N	t	1396	fixture-sheet-id
7398	2026-08-06 18:30:46.808261	160	0	0	63	0	0	\N	t	1487	fixture-sheet-id
7399	2026-08-06 18:45:46.693173	160	0	0	63	0	0	\N	t	1372	fixture-sheet-id
7400	2026-08-06 19:00:46.735744	160	0	0	63	0	0	\N	t	1414	fixture-sheet-id
7401	2026-08-06 19:15:46.838709	160	0	0	63	0	0	\N	t	1517	fixture-sheet-id
7402	2026-08-06 19:30:46.875215	160	0	0	63	0	0	\N	t	1553	fixture-sheet-id
7403	2026-08-06 19:45:46.775615	160	0	0	63	0	0	\N	t	1455	fixture-sheet-id
7404	2026-08-06 20:00:46.823297	160	0	0	63	0	0	\N	t	1502	fixture-sheet-id
7405	2026-08-06 20:15:46.679667	160	0	0	63	0	0	\N	t	1359	fixture-sheet-id
7406	2026-08-06 20:30:46.759553	160	0	0	63	0	0	\N	t	1438	fixture-sheet-id
7407	2026-08-06 20:45:46.569501	160	0	0	63	0	0	\N	t	1247	fixture-sheet-id
7408	2026-08-06 21:00:46.654209	160	0	0	63	0	0	\N	t	1334	fixture-sheet-id
7409	2026-08-06 21:15:46.764517	160	0	0	63	0	0	\N	t	1441	fixture-sheet-id
7410	2026-08-06 21:30:46.666745	160	0	0	63	0	0	\N	t	1346	fixture-sheet-id
\.


ALTER TABLE public.equipment_request_sync_log ENABLE TRIGGER ALL;

--
-- Data for Name: origins; Type: TABLE DATA; Schema: public; Owner: -
--

ALTER TABLE public.origins DISABLE TRIGGER ALL;

COPY public.origins (id, slug, label, allow_suffix, active, sort_order, created_at) FROM stdin;
1	druga-era	Druga Era	f	t	1	2026-02-23 12:45:02.164538
2	probis	Probis	f	t	2	2026-02-23 12:45:02.164538
3	netland	Netland	f	t	3	2026-02-23 12:45:02.164538
4	targowe	Targowe	f	t	4	2026-02-23 12:45:02.164538
5	personal	Personal	t	t	7	2026-02-23 12:45:02.164538
6	other	Other	t	t	8	2026-02-23 12:45:02.164538
10	innowacja	Innowacja	f	t	0	2026-06-16 14:22:20.221954
\.


ALTER TABLE public.origins ENABLE TRIGGER ALL;

--
-- Data for Name: items; Type: TABLE DATA; Schema: public; Owner: -
--

ALTER TABLE public.items DISABLE TRIGGER ALL;

COPY public.items (id, item_serial, status, location_id, item_category_id, pyr_code, origin_id, origin_suffix) FROM stdin;
3	VNC4C33551	available	1	2	PYR-PA41	2	\N
40	\N	available	1	1	PYR-LAP34	3	\N
6	VNC4D69420	available	1	2	PYR-PA44	2	\N
7	LAP11540226500	available	1	1	PYR-LAP1	3	\N
41	\N	available	1	1	PYR-LAP35	3	\N
50	\N	available	1	1	PYR-LAP44	3	\N
8	\N	available	1	1	PYR-LAP2	3	\N
44	\N	available	1	1	PYR-LAP38	3	\N
197	\N	available	1	43	PYR-T753	4	\N
47	\N	available	1	1	PYR-LAP41	3	\N
48	\N	available	1	1	PYR-LAP42	3	\N
74	0	available	1	2	PYR-PA45	2	\N
235	\N	available	1	18	PYR-TEL42	10	\N
198	\N	available	1	43	PYR-T754	4	\N
87	011	available	1	2	PYR-PA416	2	\N
77	01	available	1	2	PYR-PA46	2	\N
42	\N	available	1	1	PYR-LAP36	3	\N
207	\N	available	1	20	PYR-MIC2	1	\N
208	S7IG7D201504STM	available	1	9	PYR-ROU16	1	\N
23	\N	available	1	1	PYR-LAP17	3	\N
212	\N	available	1	20	PYR-MIC6	1	\N
20	\N	available	1	1	PYR-LAP14	3	\N
221	863314035758409	available	1	18	PYR-TEL24	1	\N
269	\N	available	1	18	PYR-TEL57	1	\N
46	\N	available	1	1	PYR-LAP40	3	\N
84	08	available	1	2	PYR-PA413	2	\N
51	\N	available	1	1	PYR-LAP45	3	\N
270	\N	available	1	18	PYR-TEL58	1	\N
5	VNC4B91585	available	1	2	PYR-PA43	2	\N
36	\N	available	1	1	PYR-LAP30	3	\N
21	\N	available	1	1	PYR-LAP15	3	\N
49	\N	available	1	1	PYR-LAP43	3	\N
97	\N	available	1	1	PYR-LAP66	2	\N
39	\N	available	1	1	PYR-LAP33	3	\N
64	\N	available	1	1	PYR-LAP58	3	\N
56	\N	available	1	1	PYR-LAP50	3	\N
94	1011	available	1	2	\N	2	\N
22	\N	available	1	1	PYR-LAP16	3	\N
95	001	available	1	2	\N	2	\N
85	09	available	1	2	PYR-PA414	2	\N
82	06	available	1	2	PYR-PA411	2	\N
89	013	available	1	2	PYR-PA418	2	\N
98	\N	available	1	1	PYR-LAP67	2	\N
99	\N	available	1	1	PYR-LAP68	2	\N
24	\N	available	1	1	PYR-LAP18	3	\N
25	\N	available	1	1	PYR-LAP19	3	\N
28	\N	available	1	1	PYR-LAP22	3	\N
289	\N	available	1	15	PYR-TVX44	10	\N
100	\N	available	1	1	PYR-LAP69	2	\N
29	\N	available	1	1	PYR-LAP23	3	\N
106	\N	available	1	1	PYR-LAP75	2	\N
43	\N	available	1	1	PYR-LAP37	3	\N
63	\N	available	1	1	PYR-LAP57	3	\N
199	\N	available	1	43	PYR-T755	4	\N
72	\N	available	17	18	PYR-TEL1	10	\N
101	\N	available	1	1	PYR-LAP70	2	\N
4	VNC3409577	available	3	2	PYR-PA42	2	\N
78	02	available	1	2	PYR-PA47	2	\N
104	\N	available	1	1	PYR-LAP73	2	\N
105	\N	available	1	1	PYR-LAP74	2	\N
107	\N	available	1	1	PYR-LAP76	2	\N
66	\N	available	1	1	PYR-LAP60	3	\N
55	\N	available	1	1	PYR-LAP49	3	\N
34	\N	available	1	1	PYR-LAP28	3	\N
45	\N	available	1	1	PYR-LAP39	3	\N
52	\N	available	1	1	PYR-LAP46	3	\N
62	\N	available	1	1	PYR-LAP56	3	\N
88	012	available	1	2	PYR-PA417	2	\N
65	\N	available	1	1	PYR-LAP59	3	\N
67	\N	available	1	1	PYR-LAP61	3	\N
68	\N	available	1	1	PYR-LAP62	3	\N
79	03	available	1	2	PYR-PA48	2	\N
54	\N	available	1	1	PYR-LAP48	3	\N
57	\N	available	1	1	PYR-LAP51	3	\N
58	\N	available	1	1	PYR-LAP52	3	\N
59	\N	available	1	1	PYR-LAP53	3	\N
60	\N	available	1	1	PYR-LAP54	3	\N
61	\N	available	1	1	PYR-LAP55	3	\N
86	010	available	1	2	PYR-PA415	2	\N
90	014	available	1	2	PYR-PA419	2	\N
19	\N	available	1	1	PYR-LAP13	3	\N
83	07	available	1	2	PYR-PA412	2	\N
288	\N	available	1	15	PYR-TVX43	10	\N
17	\N	available	1	1	PYR-LAP11	3	\N
18	\N	available	1	1	PYR-LAP12	3	\N
103	\N	available	1	1	PYR-LAP72	2	\N
30	\N	available	1	1	PYR-LAP24	3	\N
1	E02GA8202603A	available	1	7	PYR-PRO1	10	\N
2	Brak	available	1	40	PYR-Ekr1	10	\N
102	\N	available	1	1	PYR-LAP71	2	\N
297	\N	available	1	18	PYR-TEL59	1	\N
92	016	available	1	2	PYR-PA421	2	\N
209	\N	available	1	20	PYR-MIC3	1	\N
262	\N	available	1	7	PYR-PRO2	10	\N
14	\N	available	1	1	PYR-LAP8	3	\N
223	863314035792648	available	1	18	PYR-TEL9	1	\N
194	223B1N1001326	available	1	9	PYR-ROU15	1	\N
184	sdfgthjcgjg	available	1	20	PYR-MIC1	1	\N
189	2248186001619	available	1	9	PYR-ROU10	1	\N
171	\N	available	1	15	PYR-TVX26	2	\N
239	\N	available	1	18	PYR-TEL46	1	\N
172	\N	available	1	15	PYR-TVX27	2	\N
299	\N	available	1	1	PYR-LAP87	2	\N
224	863314035788729	available	1	18	PYR-TEL12	1	\N
225	863314035792689	available	1	18	PYR-TEL13	1	\N
300	\N	available	1	1	PYR-LAP88	2	\N
243	\N	available	1	6	PYR-TAB6	10	\N
227	863314035758490	available	1	18	PYR-TEL25	1	\N
183	TBIG8Y608681RAM	available	1	9	PYR-ROU6	1	\N
193	2248186001516	available	1	9	PYR-ROU14	1	\N
15	\N	available	1	1	PYR-LAP9	3	\N
230	863314035775130	available	1	18	PYR-TEL28	1	\N
238	\N	available	1	18	PYR-TEL45	10	\N
226	354293693743364	available	1	18	PYR-TEL18	1	\N
69	\N	available	1	1	PYR-LAP63	3	\N
222	352505480984873	available	17	18	PYR-TEL7	1	\N
93	017	available	1	2	PYR-PA422	2	\N
229	863314035758508	available	1	18	PYR-TEL27	1	\N
265	\N	available	1	18	PYR-TEL53	1	\N
266	\N	available	1	18	PYR-TEL54	1	\N
9	\N	available	1	1	PYR-LAP3	3	\N
16	\N	available	1	1	PYR-LAP10	3	\N
175	\N	available	1	15	PYR-TVX30	2	\N
298	\N	available	1	1	PYR-LAP86	2	\N
176	\N	available	1	15	PYR-TVX31	2	\N
166	\N	available	1	15	PYR-TVX21	2	\N
177	\N	available	1	15	PYR-TVX32	2	\N
255	\N	available	1	33	PYR-STI1	1	\N
256	\N	available	1	33	PYR-STI2	1	\N
257	\N	available	1	33	PYR-STI3	1	\N
258	\N	available	1	33	PYR-STI4	1	\N
91	015	available	1	2	PYR-PA420	2	\N
251	\N	available	1	28	PYR-GIM2	10	\N
185	TBIG8Y608317CA5	available	1	9	PYR-ROU7	1	\N
213	2350ALH053Y8	available	1	5	PYR-SPS1	1	\N
290	\N	available	1	15	PYR-TVX45	10	\N
10	\N	available	1	1	PYR-LAP4	3	\N
11	\N	available	1	1	PYR-LAP5	3	\N
169	\N	available	1	15	PYR-TVX24	2	\N
173	\N	available	1	15	PYR-TVX28	2	\N
174	\N	available	1	15	PYR-TVX29	2	\N
250	\N	available	1	28	PYR-GIM1	10	\N
271	\N	available	1	6	PYR-TAB9	10	\N
170	\N	available	1	15	PYR-TVX25	2	\N
272	\N	available	1	6	PYR-TAB10	10	\N
301	\N	available	1	1	PYR-LAP89	2	\N
302	\N	available	1	1	PYR-LAP90	2	\N
304	\N	available	1	1	PYR-LAP92	2	\N
228	863314035775122	available	1	18	PYR-TEL26	1	\N
273	\N	available	1	6	PYR-TAB11	10	\N
178	\N	available	1	15	PYR-TVX33	2	\N
274	\N	available	1	6	PYR-TAB12	10	\N
303	\N	available	1	1	PYR-LAP91	2	\N
275	\N	available	1	6	PYR-TAB13	10	\N
192	2248186001615	available	1	9	PYR-ROU13	1	\N
276	\N	available	1	6	PYR-TAB14	10	\N
277	\N	available	1	6	PYR-TAB15	10	\N
12	\N	available	1	1	PYR-LAP6	3	\N
13	\N	available	1	1	PYR-LAP7	3	\N
187	1JT1767U041E1	available	1	9	PYR-ROU8	1	\N
305	\N	available	1	28	PYR-GIM6	10	\N
195	\N	available	1	43	PYR-T751	2	\N
196	\N	available	1	43	PYR-T752	2	\N
252	\N	available	1	28	PYR-GIM3	10	\N
236	\N	available	1	18	PYR-TEL43	10	\N
138	\N	available	1	17	PYR-TV61	4	\N
139	\N	available	1	17	PYR-TV62	4	\N
145	\N	available	1	16	PYR-TV76	4	\N
237	\N	available	1	18	PYR-TEL44	10	\N
245	\N	available	1	18	PYR-TEL48	1	\N
210	\N	available	1	20	PYR-MIC4	1	\N
246	\N	available	1	18	PYR-TEL49	1	\N
231	84E95300288	available	1	2	PYR-PA423	6	zula-maskarada
247	\N	available	1	18	PYR-TEL50	1	\N
248	\N	available	1	18	PYR-TEL51	1	\N
249	\N	available	1	18	PYR-TEL52	1	\N
141	\N	available	1	16	PYR-TV72	4	\N
110	\N	available	1	1	PYR-LAP79	2	\N
216	503161070000656	available	1	5	PYR-SPS3	1	\N
111	\N	available	1	1	PYR-LAP80	2	\N
140	\N	available	1	16	PYR-TV71	4	\N
240	\N	available	1	6	PYR-TAB3	10	\N
241	\N	available	1	6	PYR-TAB4	10	\N
267	\N	available	1	18	PYR-TEL55	1	\N
108	\N	available	1	1	PYR-LAP77	2	\N
244	\N	available	1	18	PYR-TEL47	1	\N
114	\N	available	1	1	PYR-LAP83	2	\N
115	\N	available	1	1	PYR-LAP84	2	\N
259	\N	available	1	6	PYR-TAB7	10	\N
260	\N	available	1	6	PYR-TAB8	10	\N
186	\N	available	1	4	PYR-SPB1	10	\N
109	\N	available	1	1	PYR-LAP78	2	\N
190	2248186003346	available	1	9	PYR-ROU11	1	\N
144	\N	available	1	16	PYR-TV75	4	\N
142	\N	available	1	16	PYR-TV73	4	\N
143	\N	available	1	16	PYR-TV74	4	\N
180	TBIG8Y608241D4L	available	1	9	PYR-ROU3	1	\N
214	5906190445032	available	1	5	PYR-SPS2	1	\N
291	\N	available	1	21	PYR-MON16	10	\N
292	\N	available	1	21	PYR-MON17	10	\N
293	\N	available	1	21	PYR-MON18	10	\N
294	\N	available	1	21	PYR-MON19	10	\N
295	\N	available	1	21	PYR-MON20	10	\N
278	\N	available	1	6	PYR-TAB16	10	\N
112	\N	available	1	1	PYR-LAP81	2	\N
113	\N	available	1	1	PYR-LAP82	2	\N
201	\N	available	1	6	PYR-TAB1	10	\N
202	\N	available	1	6	PYR-TAB2	10	\N
117	0123	available	1	2	\N	2	\N
26	\N	available	1	1	PYR-LAP20	3	\N
165	\N	available	1	15	PYR-TVX20	4	\N
73	\N	available	17	18	PYR-TEL2	10	\N
27	\N	available	1	1	PYR-LAP21	3	\N
135	\N	available	1	21	PYR-MON13	10	\N
122	\N	available	3	9	PYR-ROU2	1	\N
156	\N	available	1	15	PYR-TVX11	4	\N
158	\N	available	1	15	PYR-TVX13	4	\N
159	\N	available	1	15	PYR-TVX14	4	\N
296	\N	available	1	4	PYR-SPB2	2	\N
160	\N	available	1	15	PYR-TVX15	4	\N
149	\N	available	1	15	PYR-TVX4	4	\N
285	\N	available	1	15	PYR-TVX40	10	\N
81	05	available	1	2	PYR-PA410	2	\N
150	\N	available	1	15	PYR-TVX5	4	\N
242	\N	available	1	6	PYR-TAB5	10	\N
232	8712581739973	available	1	18	PYR-TEL8	1	\N
217	352505480986639	available	1	18	PYR-TEL10	1	\N
188	00064F87854B	available	1	9	PYR-ROU9	1	\N
126	\N	available	1	21	PYR-MON4	10	\N
127	\N	available	1	21	PYR-MON5	10	\N
146	\N	available	1	15	PYR-TVX1	4	\N
116	\N	available	1	1	PYR-LAP85	2	\N
211	\N	available	1	20	PYR-MIC5	1	\N
121	\N	available	1	9	PYR-ROU1	1	\N
181	TBIG8Y608217GNE	available	1	9	PYR-ROU4	1	\N
254	\N	available	1	28	PYR-GIM5	10	\N
219	352505480984956	available	1	18	PYR-TEL34	1	\N
261	354293693749619	available	1	18	PYR-TEL20	1	\N
123	\N	available	1	21	PYR-MON1	10	\N
147	\N	available	1	15	PYR-TVX2	4	\N
125	\N	available	1	21	PYR-MON3	10	\N
80	04	available	1	2	PYR-PA49	2	\N
157	\N	available	1	15	PYR-TVX12	4	\N
148	\N	available	1	15	PYR-TVX3	4	\N
253	\N	available	1	28	PYR-GIM4	10	\N
124	\N	available	1	21	PYR-MON2	10	\N
233	863314035794800	available	1	18	PYR-TEL14	1	\N
70	\N	available	1	1	PYR-LAP64	3	\N
151	\N	available	1	15	PYR-TVX6	4	\N
182	TBIG8Y608234M7X	available	1	9	PYR-ROU5	1	\N
164	\N	available	1	15	PYR-TVX19	4	\N
120	\N	available	1	3	PYR-PA32	10	\N
119	\N	available	1	3	PYR-PA31	2	\N
31	\N	available	1	1	PYR-LAP25	3	\N
128	\N	available	1	21	PYR-MON6	10	\N
191	223C409001732	available	1	9	PYR-ROU12	1	\N
279	\N	available	1	15	PYR-TVX34	10	\N
71	\N	available	1	1	PYR-LAP65	3	\N
204	\N	available	1	18	PYR-TEL4	1	\N
152	\N	available	1	15	PYR-TVX7	4	\N
203	\N	available	1	18	PYR-TEL3	1	\N
129	\N	available	1	21	PYR-MON7	10	\N
130	\N	available	1	21	PYR-MON8	10	\N
132	\N	available	1	21	PYR-MON10	10	\N
133	\N	available	1	21	PYR-MON11	10	\N
283	\N	available	1	15	PYR-TVX38	10	\N
205	\N	available	1	18	PYR-TEL5	1	\N
268	\N	available	1	18	PYR-TEL56	10	\N
206	\N	available	1	18	PYR-TEL6	1	\N
218	352505481221531	available	1	18	PYR-TEL11	1	\N
286	\N	available	1	15	PYR-TVX41	10	\N
32	\N	available	1	1	PYR-LAP26	3	\N
134	\N	available	1	21	PYR-MON12	10	\N
284	\N	available	1	15	PYR-TVX39	10	\N
137	\N	available	1	21	PYR-MON15	10	\N
33	\N	available	1	1	PYR-LAP27	3	\N
136	\N	available	1	21	PYR-MON14	10	\N
35	\N	available	1	1	PYR-LAP29	3	\N
155	\N	available	1	15	PYR-TVX10	4	\N
131	\N	available	1	21	PYR-MON9	10	\N
280	\N	available	1	15	PYR-TVX35	10	\N
281	\N	available	1	15	PYR-TVX36	10	\N
220	352505481212092	available	1	18	PYR-TEL35	1	\N
287	\N	available	1	15	PYR-TVX42	10	\N
37	\N	available	1	1	PYR-LAP31	3	\N
38	\N	available	1	1	PYR-LAP32	3	\N
234	863314035792721	available	1	18	PYR-TEL15	1	\N
53	\N	available	1	1	PYR-LAP47	3	\N
282	\N	available	1	15	PYR-TVX37	10	\N
153	\N	available	1	15	PYR-TVX8	4	\N
154	\N	available	1	15	PYR-TVX9	4	\N
\.


ALTER TABLE public.items ENABLE TRIGGER ALL;

--
-- Data for Name: non_serialized_items; Type: TABLE DATA; Schema: public; Owner: -
--

ALTER TABLE public.non_serialized_items DISABLE TRIGGER ALL;

COPY public.non_serialized_items (id, item_category_id, location_id, quantity, status, origin_id, origin_suffix) FROM stdin;
79	8	11	0	\N	1	\N
39	30	38	1	\N	1	\N
40	41	38	2	\N	1	\N
6	41	33	2	\N	1	\N
7	30	33	1	\N	1	\N
22	24	34	0	\N	1	\N
43	22	34	6	\N	1	\N
16	24	1	2	\N	1	\N
10	41	3	5	\N	1	\N
71	8	32	0	\N	1	\N
20	44	22	0	\N	1	\N
85	8	27	0	\N	1	\N
26	8	2	15	\N	1	\N
18	44	1	1	\N	1	\N
3	41	1	52	\N	1	\N
13	42	1	11	\N	10	\N
11	8	33	4	\N	1	\N
84	41	27	0	\N	1	\N
50	41	17	5	\N	1	\N
14	41	51	0	\N	1	\N
12	41	31	0	\N	1	\N
120	8	56	0	\N	1	\N
51	8	17	10	\N	1	\N
55	22	1	35	\N	10	\N
19	41	22	40	\N	1	\N
21	41	34	31	\N	1	\N
23	8	34	7	\N	1	\N
27	41	2	5	\N	1	\N
117	8	55	0	\N	1	\N
36	22	1	30	\N	1	\N
56	8	52	2	\N	1	\N
93	8	16	0	\N	1	\N
121	8	22	0	\N	1	\N
58	22	22	2	\N	10	\N
59	22	24	3	\N	1	\N
92	8	20	4	\N	1	\N
61	22	22	3	\N	1	\N
70	41	8	0	\N	1	\N
35	30	31	3	\N	1	\N
115	29	43	0	\N	1	\N
89	8	43	0	\N	1	\N
90	46	43	0	\N	6	wololo
114	29	1	1	\N	1	\N
81	8	45	0	\N	1	\N
86	46	1	2137	\N	6	wololo
78	22	24	3	\N	10	\N
68	8	21	0	\N	1	\N
72	22	9	0	\N	10	\N
74	30	21	0	\N	1	\N
76	35	21	0	\N	10	\N
64	8	9	0	\N	1	\N
82	8	48	10	\N	1	\N
75	41	21	0	\N	1	\N
125	8	57	3	\N	1	\N
87	8	46	0	\N	1	\N
110	8	40	6	\N	1	\N
127	47	1	18	\N	1	\N
103	24	46	0	\N	1	\N
38	41	49	1	\N	1	\N
88	22	46	0	\N	10	\N
73	22	15	0	\N	10	\N
30	8	7	0	\N	1	\N
31	41	7	0	\N	1	\N
52	22	17	0	\N	1	\N
49	8	12	0	\N	1	\N
48	19	12	0	\N	1	\N
98	8	53	3	\N	1	\N
83	41	23	5	\N	1	\N
101	8	23	46	\N	1	\N
95	22	44	0	\N	1	\N
99	8	44	0	\N	1	\N
63	35	1	1	\N	10	\N
69	41	28	0	\N	1	\N
111	8	37	0	\N	1	\N
116	22	27	0	\N	1	\N
108	8	35	0	\N	1	\N
128	30	8	0	\N	1	\N
60	8	8	0	\N	1	\N
42	19	1	13	\N	1	\N
106	8	26	0	\N	1	\N
97	22	46	1	\N	1	\N
109	22	23	0	\N	10	\N
96	22	23	0	\N	1	\N
5	30	1	3	\N	1	\N
66	41	6	0	\N	1	\N
65	34	6	0	\N	10	\N
46	34	1	3	\N	10	\N
112	8	42	0	\N	1	\N
9	41	30	0	\N	1	\N
2	8	6	2	\N	1	\N
1	8	1	278	\N	1	\N
15	41	4	1	\N	1	\N
104	41	47	0	\N	1	\N
\.


ALTER TABLE public.non_serialized_items ENABLE TRIGGER ALL;

--
-- Data for Name: transfers; Type: TABLE DATA; Schema: public; Owner: -
--

ALTER TABLE public.transfers DISABLE TRIGGER ALL;

COPY public.transfers (id, from_location_id, to_location_id, status, transfer_date, receiver, delivery_latitude, delivery_longitude, delivery_timestamp) FROM stdin;
1	1	6	completed	2026-06-16 14:53:07.529313	\N	\N	\N	\N
2	1	30	completed	2026-06-17 10:08:43.599542	\N	\N	\N	\N
3	1	33	completed	2026-06-17 11:05:23.683103	\N	\N	\N	\N
4	1	33	completed	2026-06-17 11:41:39.354998	\N	\N	\N	\N
6	1	17	completed	2026-06-17 12:10:27.351869	\N	\N	\N	\N
5	1	30	completed	2026-06-17 11:42:38.23923	\N	\N	\N	\N
9	1	3	completed	2026-06-17 13:09:41.788984	\N	\N	\N	\N
10	1	33	completed	2026-06-17 13:30:47.691199	\N	\N	\N	\N
12	1	31	completed	2026-06-17 13:44:41.003492	\N	\N	\N	\N
59	17	1	completed	2026-06-18 09:02:28.323218	\N	\N	\N	\N
54	1	4	cancelled	2026-06-18 08:22:06.7819	\N	\N	\N	\N
63	1	52	completed	2026-06-18 09:57:16.91167	\N	\N	\N	\N
61	1	22	completed	2026-06-18 09:24:18.417537	\N	\N	\N	\N
23	1	51	completed	2026-06-17 13:58:42.52508	\N	\N	\N	\N
11	1	30	completed	2026-06-17 13:39:20.102903	\N	\N	\N	\N
7	1	4	completed	2026-06-17 13:00:51.089519	\N	\N	\N	\N
8	1	4	cancelled	2026-06-17 13:03:45.718702	\N	\N	\N	\N
27	31	1	completed	2026-06-17 14:37:18.505622	\N	\N	\N	\N
28	1	22	completed	2026-06-17 15:15:01.129514	\N	\N	\N	\N
24	1	34	completed	2026-06-17 14:17:15.16307	\N	\N	\N	\N
25	1	34	completed	2026-06-17 14:30:48.748881	\N	\N	\N	\N
26	1	34	completed	2026-06-17 14:33:10.517207	\N	\N	\N	\N
31	1	34	completed	2026-06-17 15:47:04.318231	\N	\N	\N	\N
29	1	2	completed	2026-06-17 15:30:31.383371	\N	\N	\N	\N
30	1	45	completed	2026-06-17 15:43:54.091817	\N	\N	\N	\N
32	1	30	cancelled	2026-06-17 15:56:39.806807	\N	\N	\N	\N
33	1	2	completed	2026-06-17 17:18:20.946713	\N	\N	\N	\N
34	1	7	completed	2026-06-17 17:36:45.785808	\N	\N	\N	\N
35	7	1	completed	2026-06-17 19:18:00.649809	\N	\N	\N	\N
37	1	30	cancelled	2026-06-18 07:09:39.150591	\N	\N	\N	\N
49	1	31	completed	2026-06-18 07:45:46.696816	\N	\N	\N	\N
46	1	13	completed	2026-06-18 07:33:37.04908	\N	\N	\N	\N
44	1	37	cancelled	2026-06-18 07:29:30.0686	\N	\N	\N	\N
39	1	49	completed	2026-06-18 07:17:39.334746	\N	\N	\N	\N
36	1	49	completed	2026-06-18 07:07:13.568777	\N	\N	\N	\N
38	1	30	completed	2026-06-18 07:11:47.474339	\N	\N	\N	\N
40	1	30	completed	2026-06-18 07:18:41.837789	\N	\N	\N	\N
41	1	38	completed	2026-06-18 07:20:35.685211	\N	\N	\N	\N
42	1	30	completed	2026-06-18 07:25:18.202157	\N	\N	\N	\N
43	1	30	completed	2026-06-18 07:27:30.133208	\N	\N	\N	\N
45	1	30	completed	2026-06-18 07:30:31.071197	\N	\N	\N	\N
51	1	34	completed	2026-06-18 07:58:34.279868	\N	\N	\N	\N
47	1	35	cancelled	2026-06-18 07:37:30.724606	\N	\N	\N	\N
50	1	35	cancelled	2026-06-18 07:47:33.730765	\N	\N	\N	\N
48	1	14	completed	2026-06-18 07:39:23.235766	\N	\N	\N	\N
53	1	2	completed	2026-06-18 08:19:11.918004	\N	\N	\N	\N
52	1	12	completed	2026-06-18 08:14:57.76653	\N	\N	\N	\N
56	1	17	completed	2026-06-18 08:48:58.931948	\N	\N	\N	\N
58	1	17	completed	2026-06-18 08:59:24.32974	\N	\N	\N	\N
57	1	17	completed	2026-06-18 08:51:15.183917	\N	\N	\N	\N
60	1	24	completed	2026-06-18 09:23:18.491588	\N	\N	\N	\N
62	1	8	completed	2026-06-18 09:45:15.096941	\N	\N	\N	\N
64	1	22	completed	2026-06-18 09:59:25.132762	\N	\N	\N	\N
65	1	9	completed	2026-06-18 10:10:22.480975	\N	\N	\N	\N
55	1	6	completed	2026-06-18 08:41:55.250712	\N	\N	\N	\N
66	1	21	completed	2026-06-18 10:41:19.19366	\N	\N	\N	\N
70	1	28	completed	2026-06-18 11:27:06.450147	\N	\N	\N	\N
71	1	8	completed	2026-06-18 11:27:35.467469	\N	\N	\N	\N
72	1	32	completed	2026-06-18 11:40:34.932435	\N	\N	\N	\N
69	1	9	completed	2026-06-18 10:56:55.0534	\N	\N	\N	\N
80	1	15	completed	2026-06-18 12:15:03.905734	\N	\N	\N	\N
67	1	21	completed	2026-06-18 10:44:07.678123	\N	\N	\N	\N
68	1	21	completed	2026-06-18 10:46:51.158666	\N	\N	\N	\N
73	1	15	completed	2026-06-18 11:45:31.60872	\N	\N	\N	\N
87	1	24	completed	2026-06-18 13:18:13.168331	\N	\N	\N	\N
90	1	24	completed	2026-06-18 13:32:48.823754	\N	\N	\N	\N
88	1	6	completed	2026-06-18 13:24:42.30162	\N	\N	\N	\N
89	1	11	completed	2026-06-18 13:31:09.41008	\N	\N	\N	\N
91	1	45	completed	2026-06-18 13:34:57.921371	\N	\N	\N	\N
92	1	45	completed	2026-06-18 13:38:23.051149	\N	\N	\N	\N
93	1	48	completed	2026-06-18 13:40:24.171009	\N	\N	\N	\N
94	1	23	completed	2026-06-18 13:48:53.696668	\N	\N	\N	\N
95	1	27	completed	2026-06-18 13:48:58.098568	\N	\N	\N	\N
97	1	27	completed	2026-06-18 14:16:01.205068	\N	\N	\N	\N
100	1	46	completed	2026-06-18 14:24:52.968779	\N	\N	\N	\N
99	1	46	completed	2026-06-18 14:22:50.933844	\N	\N	\N	\N
98	1	46	completed	2026-06-18 14:20:46.848225	\N	\N	\N	\N
101	1	43	completed	2026-06-18 14:33:32.069788	\N	\N	\N	\N
102	1	43	completed	2026-06-18 14:35:04.435316	\N	\N	\N	\N
96	1	27	completed	2026-06-18 14:05:43.697709	\N	\N	\N	\N
105	1	20	completed	2026-06-18 14:59:24.481099	\N	\N	\N	\N
104	1	16	completed	2026-06-18 14:52:08.543041	\N	\N	\N	\N
109	1	46	completed	2026-06-18 16:24:25.181174	\N	\N	\N	\N
112	1	44	completed	2026-06-18 16:31:16.011991	\N	\N	\N	\N
111	1	23	completed	2026-06-18 16:30:49.648448	\N	\N	\N	\N
110	1	46	completed	2026-06-18 16:30:28.634509	\N	\N	\N	\N
113	1	53	completed	2026-06-18 16:42:46.496851	\N	\N	\N	\N
114	1	53	completed	2026-06-18 16:43:18.631901	\N	\N	\N	\N
106	1	44	completed	2026-06-18 15:38:49.783532	\N	\N	\N	\N
107	1	23	completed	2026-06-18 15:50:42.118485	\N	\N	\N	\N
108	1	46	completed	2026-06-18 15:52:36.017938	\N	\N	\N	\N
103	1	2	completed	2026-06-18 14:43:20.714757	\N	\N	\N	\N
115	1	23	completed	2026-06-18 16:50:20.023206	\N	\N	\N	\N
116	1	44	completed	2026-06-18 16:51:01.066875	\N	\N	\N	\N
118	2	1	completed	2026-06-18 17:14:25.063741	\N	\N	\N	\N
119	1	47	completed	2026-06-18 17:47:00.169724	\N	\N	\N	\N
117	1	23	completed	2026-06-18 17:04:52.876195	\N	\N	\N	\N
120	1	26	completed	2026-06-18 18:21:28.551153	\N	\N	\N	\N
121	1	33	completed	2026-06-18 19:17:06.058477	\N	\N	\N	\N
122	6	1	completed	2026-06-18 20:13:06.717597	\N	\N	\N	\N
124	1	2	completed	2026-06-19 06:24:09.740186	\N	\N	\N	\N
126	1	35	completed	2026-06-19 06:25:41.674531	\N	\N	\N	\N
125	2	1	completed	2026-06-19 06:24:57.703435	\N	\N	\N	\N
123	1	23	completed	2026-06-19 06:21:43.642231	\N	\N	\N	\N
133	1	40	completed	2026-06-19 07:44:34.989042	\N	\N	\N	\N
127	1	34	completed	2026-06-19 07:07:54.190858	\N	\N	\N	\N
134	1	17	completed	2026-06-19 08:25:46.281769	\N	\N	\N	\N
132	1	55	completed	2026-06-19 07:39:44.868241	\N	\N	\N	\N
131	1	6	completed	2026-06-19 07:35:05.220889	\N	\N	\N	\N
128	1	16	completed	2026-06-19 07:30:54.815517	\N	\N	\N	\N
130	1	14	completed	2026-06-19 07:33:51.812271	\N	\N	\N	\N
135	17	1	completed	2026-06-19 08:29:15.230897	\N	\N	\N	\N
136	1	37	completed	2026-06-19 08:35:40.415837	\N	\N	\N	\N
137	1	37	completed	2026-06-19 08:49:29.747335	\N	\N	\N	\N
138	1	42	completed	2026-06-19 08:50:49.761572	\N	\N	\N	\N
139	1	35	completed	2026-06-19 08:51:30.10273	\N	\N	\N	\N
142	27	1	completed	2026-06-19 09:22:37.415778	\N	\N	\N	\N
143	1	43	completed	2026-06-19 09:28:07.191985	\N	\N	\N	\N
141	1	27	completed	2026-06-19 09:11:27.48584	\N	\N	\N	\N
140	1	27	completed	2026-06-19 09:03:07.535707	\N	\N	\N	\N
129	1	20	completed	2026-06-19 07:32:43.209681	\N	\N	\N	\N
144	1	6	completed	2026-06-19 09:43:08.749739	\N	\N	\N	\N
145	1	11	completed	2026-06-19 10:02:41.162741	\N	\N	\N	\N
146	1	56	completed	2026-06-19 10:16:17.758573	\N	\N	\N	\N
149	1	22	completed	2026-06-19 11:55:11.792795	\N	\N	\N	\N
151	1	26	completed	2026-06-19 13:40:30.008781	\N	\N	\N	\N
152	27	1	completed	2026-06-19 13:41:35.675621	\N	\N	\N	\N
153	9	1	completed	2026-06-19 13:42:06.156769	\N	\N	\N	\N
154	1	25	completed	2026-06-19 14:46:27.836366	\N	\N	\N	\N
150	1	9	completed	2026-06-19 12:23:35.496706	\N	\N	\N	\N
147	1	21	completed	2026-06-19 11:02:51.152731	\N	\N	\N	\N
148	1	57	completed	2026-06-19 11:09:42.033342	\N	\N	\N	\N
157	1	40	completed	2026-06-19 18:44:21.903154	\N	\N	\N	\N
156	1	20	completed	2026-06-19 18:10:01.691215	\N	\N	\N	\N
155	1	8	completed	2026-06-19 14:56:50.259756	\N	\N	\N	\N
158	11	1	completed	2026-06-20 09:38:36.18794	\N	\N	\N	\N
161	6	42	completed	2026-06-20 15:27:38.366717	\N	\N	\N	\N
162	20	1	completed	2026-06-20 15:39:25.285459	\N	\N	\N	\N
163	52	1	completed	2026-06-20 17:39:05.223599	\N	\N	\N	\N
164	1	17	completed	2026-06-20 17:41:43.733139	\N	\N	\N	\N
165	17	1	completed	2026-06-20 17:42:08.998389	\N	\N	\N	\N
166	25	1	completed	2026-06-20 19:19:50.87224	\N	\N	\N	\N
159	1	11	completed	2026-06-20 09:40:06.126983	\N	\N	\N	\N
160	1	27	cancelled	2026-06-20 15:11:50.295267	\N	\N	\N	\N
167	44	1	completed	2026-06-21 11:55:25.987618	\N	\N	\N	\N
168	12	1	completed	2026-06-21 12:31:11.413342	\N	\N	\N	\N
169	30	1	completed	2026-06-21 12:34:07.582822	\N	\N	\N	\N
170	49	1	completed	2026-06-21 12:53:24.474737	\N	\N	\N	\N
171	30	1	completed	2026-06-21 12:57:31.64295	\N	\N	\N	\N
172	15	1	completed	2026-06-21 12:58:00.467732	\N	\N	\N	\N
173	17	1	completed	2026-06-21 13:23:52.554831	\N	\N	\N	\N
174	44	1	completed	2026-06-21 13:32:18.709062	\N	\N	\N	\N
175	23	1	completed	2026-06-21 13:34:58.238141	\N	\N	\N	\N
176	49	1	completed	2026-06-21 14:02:23.855078	\N	\N	\N	\N
177	28	1	completed	2026-06-21 14:03:41.890614	\N	\N	\N	\N
178	53	1	completed	2026-06-21 14:04:29.108696	\N	\N	\N	\N
179	33	1	completed	2026-06-21 14:19:37.623902	\N	\N	\N	\N
180	27	1	completed	2026-06-21 14:27:56.264122	\N	\N	\N	\N
181	27	1	completed	2026-06-21 14:29:16.032977	\N	\N	\N	\N
182	53	1	completed	2026-06-21 14:30:24.323711	\N	\N	\N	\N
183	11	1	completed	2026-06-21 14:38:21.229418	\N	\N	\N	\N
184	11	1	completed	2026-06-21 14:39:21.736513	\N	\N	\N	\N
185	27	1	completed	2026-06-21 14:40:43.207171	\N	\N	\N	\N
186	27	1	completed	2026-06-21 14:42:14.776997	\N	\N	\N	\N
187	27	1	completed	2026-06-21 14:43:14.499875	\N	\N	\N	\N
188	23	1	completed	2026-06-21 14:44:35.335108	\N	\N	\N	\N
189	27	1	completed	2026-06-21 14:45:42.966524	\N	\N	\N	\N
190	27	1	completed	2026-06-21 14:47:07.663786	\N	\N	\N	\N
191	56	1	completed	2026-06-21 15:00:00.695977	\N	\N	\N	\N
192	26	1	completed	2026-06-21 15:03:55.48943	\N	\N	\N	\N
193	55	1	completed	2026-06-21 15:08:12.380414	\N	\N	\N	\N
194	16	1	completed	2026-06-21 15:12:23.771214	\N	\N	\N	\N
195	22	1	completed	2026-06-21 15:17:38.013227	\N	\N	\N	\N
196	43	1	completed	2026-06-21 15:20:39.637825	\N	\N	\N	\N
197	21	1	completed	2026-06-21 15:22:13.494261	\N	\N	\N	\N
198	46	1	completed	2026-06-21 15:23:38.669195	\N	\N	\N	\N
199	12	1	completed	2026-06-21 15:27:25.660309	\N	\N	\N	\N
200	6	1	completed	2026-06-21 15:35:34.630316	\N	\N	\N	\N
201	27	1	completed	2026-06-21 15:47:29.253796	\N	\N	\N	\N
202	26	1	completed	2026-06-21 15:49:27.284495	\N	\N	\N	\N
203	46	1	completed	2026-06-21 15:50:42.508806	\N	\N	\N	\N
205	8	1	completed	2026-06-21 16:05:55.554211	\N	\N	\N	\N
206	32	1	completed	2026-06-21 16:19:08.172369	\N	\N	\N	\N
207	13	1	completed	2026-06-21 16:26:02.928183	\N	\N	\N	\N
208	30	1	completed	2026-06-21 16:27:35.660836	\N	\N	\N	\N
204	23	1	completed	2026-06-21 15:59:10.740722	\N	\N	\N	\N
209	46	1	completed	2026-06-21 16:35:51.010965	\N	\N	\N	\N
210	20	1	completed	2026-06-21 16:39:53.929164	\N	\N	\N	\N
211	8	1	completed	2026-06-21 16:42:28.988446	\N	\N	\N	\N
212	45	1	completed	2026-06-21 16:55:46.218624	\N	\N	\N	\N
213	2	1	completed	2026-06-21 17:12:08.276987	\N	\N	\N	\N
214	6	1	completed	2026-06-21 17:14:15.851588	\N	\N	\N	\N
215	9	1	completed	2026-06-21 17:16:17.909207	\N	\N	\N	\N
216	30	1	completed	2026-06-21 17:16:54.832541	\N	\N	\N	\N
217	44	1	completed	2026-06-21 17:20:50.808799	\N	\N	\N	\N
218	21	1	completed	2026-06-21 17:28:16.445202	\N	\N	\N	\N
219	7	1	completed	2026-06-21 17:29:28.032857	\N	\N	\N	\N
220	37	1	completed	2026-06-21 17:32:03.777371	\N	\N	\N	\N
221	46	1	completed	2026-06-21 17:34:11.196462	\N	\N	\N	\N
222	35	1	completed	2026-06-21 17:35:22.374318	\N	\N	\N	\N
223	17	1	completed	2026-06-21 17:38:07.040413	\N	\N	\N	\N
224	14	1	completed	2026-06-21 17:38:47.730184	\N	\N	\N	\N
225	30	1	completed	2026-06-21 17:39:09.003394	\N	\N	\N	\N
226	14	1	completed	2026-06-21 17:39:41.91978	\N	\N	\N	\N
227	30	1	completed	2026-06-21 17:40:14.116089	\N	\N	\N	\N
228	8	1	completed	2026-06-21 17:45:29.533846	\N	\N	\N	\N
229	6	1	completed	2026-06-21 17:50:51.626698	\N	\N	\N	\N
230	17	1	completed	2026-06-21 17:59:29.963749	\N	\N	\N	\N
231	42	1	completed	2026-06-21 18:06:57.145201	\N	\N	\N	\N
232	17	1	completed	2026-06-21 18:10:42.440555	\N	\N	\N	\N
233	51	1	completed	2026-06-21 18:12:12.515566	\N	\N	\N	\N
234	17	1	completed	2026-06-21 18:13:33.351466	\N	\N	\N	\N
235	30	1	completed	2026-06-21 18:20:26.776419	\N	\N	\N	\N
236	34	1	completed	2026-06-21 18:23:44.307317	\N	\N	\N	\N
237	30	1	completed	2026-06-21 18:24:08.367919	\N	\N	\N	\N
238	23	1	completed	2026-06-21 18:25:45.590962	\N	\N	\N	\N
239	27	1	completed	2026-06-21 18:26:08.539	\N	\N	\N	\N
240	4	1	completed	2026-06-21 18:45:32.848566	\N	\N	\N	\N
241	6	1	completed	2026-06-21 18:45:43.301391	\N	\N	\N	\N
242	4	1	completed	2026-06-21 18:46:04.798981	\N	\N	\N	\N
244	17	1	completed	2026-06-21 18:51:03.779164	\N	\N	\N	\N
245	30	1	completed	2026-06-21 18:52:55.169957	\N	\N	\N	\N
243	47	1	completed	2026-06-21 18:49:33.515817	\N	\N	\N	\N
246	34	1	completed	2026-06-21 18:53:19.080701	\N	\N	\N	\N
247	22	1	completed	2026-06-21 18:54:07.514251	\N	\N	\N	\N
248	24	1	completed	2026-06-21 18:55:13.571896	\N	\N	\N	\N
250	34	1	completed	2026-06-21 19:39:30.513343	\N	\N	\N	\N
251	34	1	completed	2026-06-21 19:41:36.122831	\N	\N	\N	\N
252	33	1	completed	2026-06-22 08:59:45.796196	\N	\N	\N	\N
253	42	1	completed	2026-06-22 09:44:25.20559	\N	\N	\N	\N
254	22	1	completed	2026-06-22 09:44:52.019382	\N	\N	\N	\N
249	24	1	completed	2026-06-21 19:31:17.922701	\N	\N	\N	\N
255	22	1	completed	2026-06-22 10:09:59.385073	\N	\N	\N	\N
256	34	1	completed	2026-06-22 10:10:16.921859	\N	\N	\N	\N
257	51	1	in_transit	2026-06-22 16:08:00.900445	\N	\N	\N	\N
\.


ALTER TABLE public.transfers ENABLE TRIGGER ALL;

--
-- Data for Name: non_serialized_transfers; Type: TABLE DATA; Schema: public; Owner: -
--

ALTER TABLE public.non_serialized_transfers DISABLE TRIGGER ALL;

COPY public.non_serialized_transfers (id, transfer_id, item_category_id, quantity, stock_id, status, origin_id, origin_suffix) FROM stdin;
1	1	8	1	1	completed	1	\N
2	3	41	2	3	completed	1	\N
3	3	30	1	5	completed	1	\N
4	5	41	1	3	completed	1	\N
6	9	41	5	3	completed	1	\N
7	10	8	3	1	completed	1	\N
8	12	41	3	3	completed	1	\N
19	23	41	1	3	completed	1	\N
5	7	41	5	3	completed	1	\N
25	27	41	3	12	completed	1	\N
26	28	41	40	3	completed	1	\N
27	28	44	1	18	completed	1	\N
20	24	41	6	3	completed	1	\N
21	25	24	1	16	completed	1	\N
22	25	8	1	1	completed	1	\N
23	25	41	25	3	completed	1	\N
24	26	8	6	1	completed	1	\N
28	29	8	5	1	completed	1	\N
29	29	41	5	3	completed	1	\N
30	32	41	4	3	cancelled	1	\N
31	33	8	10	1	completed	1	\N
32	34	8	20	1	completed	1	\N
33	34	41	10	3	completed	1	\N
34	35	8	1	30	completed	1	\N
35	37	41	4	3	cancelled	1	\N
42	49	30	3	5	completed	1	\N
40	44	41	4	3	cancelled	1	\N
36	39	41	3	3	completed	1	\N
37	41	30	1	5	completed	1	\N
38	41	41	2	3	completed	1	\N
39	43	41	4	3	completed	1	\N
44	51	22	6	36	completed	1	\N
41	47	41	4	3	cancelled	1	\N
43	50	8	2	1	cancelled	1	\N
47	53	8	10	1	completed	1	\N
45	52	19	11	42	completed	1	\N
46	52	8	15	1	completed	1	\N
51	56	41	5	3	completed	1	\N
52	56	8	10	1	completed	1	\N
54	58	22	1	36	completed	1	\N
53	57	8	15	1	completed	1	\N
55	59	8	15	51	completed	1	\N
59	63	8	2	1	completed	1	\N
57	61	22	2	55	completed	10	\N
56	60	22	3	36	completed	1	\N
58	62	8	18	1	completed	1	\N
60	64	22	3	36	completed	1	\N
61	65	8	40	1	completed	1	\N
48	55	34	3	46	completed	10	\N
49	55	41	5	3	completed	1	\N
50	55	8	21	1	completed	1	\N
62	66	8	15	1	completed	1	\N
67	70	41	5	3	completed	1	\N
68	71	41	10	3	completed	1	\N
69	72	8	1	1	completed	1	\N
66	69	22	4	55	completed	10	\N
76	80	22	1	55	completed	10	\N
63	67	30	1	5	completed	1	\N
64	67	41	1	3	completed	1	\N
65	68	35	1	63	completed	10	\N
83	90	22	3	55	completed	10	\N
82	89	8	4	1	completed	1	\N
84	92	8	3	1	completed	1	\N
85	93	8	10	1	completed	1	\N
86	94	41	1	3	completed	1	\N
87	95	41	2	3	completed	1	\N
88	97	8	10	1	completed	1	\N
90	100	8	60	1	completed	1	\N
89	99	22	3	55	completed	10	\N
91	101	8	15	1	completed	1	\N
92	102	46	1	86	completed	6	wololo
94	105	8	30	1	completed	1	\N
93	104	8	3	1	completed	1	\N
102	112	22	1	36	completed	1	\N
101	111	22	7	36	completed	1	\N
100	110	22	4	36	completed	1	\N
103	114	8	3	1	completed	1	\N
95	106	8	4	1	completed	1	\N
96	107	41	4	3	completed	1	\N
97	107	8	46	1	completed	1	\N
98	108	8	1	1	completed	1	\N
99	108	24	1	16	completed	1	\N
105	119	41	2	3	completed	1	\N
104	117	22	8	36	completed	1	\N
106	120	8	1	1	completed	1	\N
107	121	8	1	1	completed	1	\N
109	126	8	3	1	completed	1	\N
108	123	22	5	55	completed	10	\N
111	133	8	5	1	completed	1	\N
112	136	8	2	1	completed	1	\N
113	138	8	2	1	completed	1	\N
114	139	8	2	1	completed	1	\N
116	143	29	1	114	completed	1	\N
115	141	22	2	36	completed	1	\N
110	132	8	3	1	completed	1	\N
117	144	8	3	1	completed	1	\N
118	145	8	1	1	completed	1	\N
119	146	8	1	1	completed	1	\N
121	149	8	1	1	completed	1	\N
120	148	8	3	1	completed	1	\N
123	153	22	4	72	completed	10	\N
122	150	8	3	1	completed	1	\N
125	157	8	1	1	completed	1	\N
124	155	30	2	5	completed	1	\N
126	170	41	2	38	completed	1	\N
127	172	22	1	73	completed	10	\N
128	173	22	1	52	completed	1	\N
129	174	22	1	95	completed	1	\N
130	174	8	4	99	completed	1	\N
131	177	41	5	69	completed	1	\N
132	180	8	4	85	completed	1	\N
133	180	22	2	116	completed	1	\N
134	183	8	5	79	completed	1	\N
135	186	8	6	85	completed	1	\N
136	189	41	2	84	completed	1	\N
137	191	8	1	120	completed	1	\N
138	193	8	3	117	completed	1	\N
139	194	8	3	93	completed	1	\N
140	195	8	1	121	completed	1	\N
141	196	29	1	115	completed	1	\N
142	196	8	15	89	completed	1	\N
143	196	46	1	90	completed	6	wololo
144	197	8	15	68	completed	1	\N
145	197	30	1	74	completed	1	\N
146	197	35	1	76	completed	10	\N
147	197	41	1	75	completed	1	\N
148	198	8	61	87	completed	1	\N
149	198	24	1	103	completed	1	\N
150	198	22	3	88	completed	10	\N
151	199	8	15	49	completed	1	\N
152	199	19	11	48	completed	1	\N
153	202	8	1	106	completed	1	\N
154	203	22	3	97	completed	1	\N
157	205	30	1	128	completed	1	\N
158	206	8	1	71	completed	1	\N
159	208	41	4	9	completed	1	\N
155	204	22	5	109	completed	10	\N
156	204	22	15	96	completed	1	\N
160	210	8	26	92	completed	1	\N
161	211	41	10	70	completed	1	\N
162	212	8	3	81	completed	1	\N
163	215	8	43	64	completed	1	\N
164	219	8	19	30	completed	1	\N
165	219	41	10	31	completed	1	\N
166	220	8	2	111	completed	1	\N
167	222	8	4	108	completed	1	\N
168	228	30	1	128	completed	1	\N
169	228	8	18	60	completed	1	\N
170	229	41	5	66	completed	1	\N
171	229	34	3	65	completed	10	\N
172	229	8	22	2	completed	1	\N
173	231	8	2	112	completed	1	\N
174	237	41	1	9	completed	1	\N
175	240	41	3	15	completed	1	\N
176	241	8	1	2	completed	1	\N
177	242	41	1	15	completed	1	\N
179	246	24	1	22	completed	1	\N
180	247	44	1	20	completed	1	\N
178	243	41	2	104	completed	1	\N
181	257	41	1	14	in_transit	1	\N
\.


ALTER TABLE public.non_serialized_transfers ENABLE TRIGGER ALL;

--
-- Data for Name: pyr_code_reservations; Type: TABLE DATA; Schema: public; Owner: -
--

ALTER TABLE public.pyr_code_reservations DISABLE TRIGGER ALL;

COPY public.pyr_code_reservations (id, pyr_code, category_id, reserved_at, claimed_at, item_id) FROM stdin;
1	PYR-PA41	2	2026-06-17 09:55:32.39988+00	2026-06-17 10:04:28.853937+00	3
2	PYR-PA42	2	2026-06-17 09:55:32.420979+00	2026-06-17 10:25:30.442057+00	4
3	PYR-PA43	2	2026-06-17 09:55:32.426242+00	2026-06-17 10:27:36.041952+00	5
4	PYR-PA44	2	2026-06-17 09:55:32.430664+00	2026-06-17 10:29:50.978958+00	6
6	PYR-PA46	2	2026-06-17 09:55:32.440207+00	2026-06-17 12:44:10.598136+00	77
7	PYR-PA47	2	2026-06-17 09:55:32.444934+00	2026-06-17 12:44:10.613734+00	78
8	PYR-PA48	2	2026-06-17 09:55:32.449145+00	2026-06-17 12:44:10.620663+00	79
9	PYR-PA49	2	2026-06-17 09:55:32.453766+00	2026-06-17 12:44:10.62779+00	80
10	PYR-PA410	2	2026-06-17 09:55:32.460484+00	2026-06-17 12:44:10.636678+00	81
11	PYR-PA411	2	2026-06-17 09:55:32.465021+00	2026-06-17 12:44:10.646211+00	82
12	PYR-PA412	2	2026-06-17 09:55:32.470711+00	2026-06-17 12:44:10.656804+00	83
13	PYR-PA413	2	2026-06-17 09:55:32.475727+00	2026-06-17 12:44:10.667598+00	84
14	PYR-PA414	2	2026-06-17 09:55:32.482729+00	2026-06-17 12:44:10.686521+00	85
15	PYR-PA415	2	2026-06-17 09:55:32.488556+00	2026-06-17 12:44:10.701222+00	86
16	PYR-PA416	2	2026-06-17 09:55:32.496115+00	2026-06-17 12:44:10.71122+00	87
17	PYR-PA417	2	2026-06-17 09:55:32.503145+00	2026-06-17 12:44:10.729949+00	88
18	PYR-PA418	2	2026-06-17 09:55:32.511683+00	2026-06-17 12:44:10.737035+00	89
19	PYR-PA419	2	2026-06-17 09:55:32.517149+00	2026-06-17 12:44:10.744995+00	90
20	PYR-PA420	2	2026-06-17 09:55:32.522589+00	2026-06-17 12:44:10.756475+00	91
21	PYR-PA421	2	2026-06-17 09:55:32.529784+00	2026-06-17 12:44:10.766894+00	92
22	PYR-PA422	2	2026-06-17 09:55:32.534708+00	2026-06-17 12:44:10.774473+00	93
32	PYR-TEL16	18	2026-06-17 16:43:23.61602+00	\N	\N
33	PYR-TEL17	18	2026-06-17 16:43:23.621752+00	\N	\N
35	PYR-TEL19	18	2026-06-17 16:43:23.638321+00	\N	\N
37	PYR-TEL21	18	2026-06-17 16:43:23.65293+00	\N	\N
38	PYR-TEL22	18	2026-06-17 16:43:23.659781+00	\N	\N
39	PYR-TEL23	18	2026-06-17 16:43:23.666125+00	\N	\N
45	PYR-TEL29	18	2026-06-17 16:43:23.708523+00	\N	\N
46	PYR-TEL30	18	2026-06-17 16:43:23.713288+00	\N	\N
47	PYR-TEL31	18	2026-06-17 16:43:23.719739+00	\N	\N
48	PYR-TEL32	18	2026-06-17 16:43:23.727224+00	\N	\N
49	PYR-TEL33	18	2026-06-17 16:43:23.73496+00	\N	\N
52	PYR-TEL36	18	2026-06-17 16:43:23.757125+00	\N	\N
53	PYR-TEL37	18	2026-06-17 16:43:23.763734+00	\N	\N
54	PYR-TEL38	18	2026-06-17 16:43:23.771713+00	\N	\N
55	PYR-TEL39	18	2026-06-17 16:43:23.776511+00	\N	\N
56	PYR-TEL40	18	2026-06-17 16:43:23.781044+00	\N	\N
57	PYR-TEL41	18	2026-06-17 16:43:23.785783+00	\N	\N
26	PYR-TEL10	18	2026-06-17 16:43:23.584964+00	2026-06-17 16:48:00.114775+00	217
27	PYR-TEL11	18	2026-06-17 16:43:23.590392+00	2026-06-17 16:48:00.150724+00	218
50	PYR-TEL34	18	2026-06-17 16:43:23.741723+00	2026-06-17 16:48:00.172203+00	219
51	PYR-TEL35	18	2026-06-17 16:43:23.749312+00	2026-06-17 16:48:00.197322+00	220
40	PYR-TEL24	18	2026-06-17 16:43:23.677449+00	2026-06-17 17:05:18.3235+00	221
23	PYR-TEL7	18	2026-06-17 16:43:23.570299+00	2026-06-17 17:28:58.263386+00	222
25	PYR-TEL9	18	2026-06-17 16:43:23.580482+00	2026-06-17 17:28:58.273579+00	223
28	PYR-TEL12	18	2026-06-17 16:43:23.594825+00	2026-06-17 17:28:58.279962+00	224
29	PYR-TEL13	18	2026-06-17 16:43:23.60043+00	2026-06-17 17:28:58.286745+00	225
34	PYR-TEL18	18	2026-06-17 16:43:23.629734+00	2026-06-17 17:28:58.292866+00	226
41	PYR-TEL25	18	2026-06-17 16:43:23.682915+00	2026-06-17 17:28:58.298623+00	227
42	PYR-TEL26	18	2026-06-17 16:43:23.690144+00	2026-06-17 17:28:58.303913+00	228
43	PYR-TEL27	18	2026-06-17 16:43:23.696302+00	2026-06-17 17:28:58.309308+00	229
44	PYR-TEL28	18	2026-06-17 16:43:23.700654+00	2026-06-17 17:28:58.31442+00	230
24	PYR-TEL8	18	2026-06-17 16:43:23.576017+00	2026-06-17 17:41:13.136686+00	232
30	PYR-TEL14	18	2026-06-17 16:43:23.606733+00	2026-06-17 17:41:13.142946+00	233
31	PYR-TEL15	18	2026-06-17 16:43:23.611261+00	2026-06-17 17:41:13.149904+00	234
36	PYR-TEL20	18	2026-06-17 16:43:23.645925+00	2026-06-18 10:40:38.138404+00	261
\.


ALTER TABLE public.pyr_code_reservations ENABLE TRIGGER ALL;

--
-- Data for Name: quest_transfers; Type: TABLE DATA; Schema: public; Owner: -
--

ALTER TABLE public.quest_transfers DISABLE TRIGGER ALL;

COPY public.quest_transfers (id, quest_id, transfer_id, created_at) FROM stdin;
1	quest-dd264c46a8148ea8	1	2026-06-16 14:53:07.542306
2	quest-89897f90580d4fd2	2	2026-06-17 10:08:43.615093
3	quest-e2acfdd55cc905b5	6	2026-06-17 12:10:27.369351
4	quest-5fcd84b32e964c86	7	2026-06-17 13:00:51.101531
5	quest-f1f5fa761ea39cb5	9	2026-06-17 13:09:41.815577
6	quest-df25a356d916ca48	10	2026-06-17 13:30:47.70052
7	quest-537322d885cce98c	11	2026-06-17 13:39:20.113097
8	quest-a33bf6882535a57c	23	2026-06-17 13:58:42.539931
9	quest-aeb9ae30df1e240f	24	2026-06-17 14:17:15.17812
10	quest-173deb2712ce5d96	25	2026-06-17 14:30:48.764931
11	quest-aeb9ae30df1e240f	26	2026-06-17 14:33:10.525998
12	quest-b2e7db8422bd567a	28	2026-06-17 15:15:01.181705
13	quest-458151fb13775fb8	29	2026-06-17 15:30:31.399882
14	quest-422eae9ff39e977c	30	2026-06-17 15:43:54.100601
15	quest-6bc0b7ba43cba386	31	2026-06-17 15:47:04.342669
17	quest-458151fb13775fb8	33	2026-06-17 17:18:20.958425
18	quest-6ad1d4815d784c36	34	2026-06-17 17:36:45.797504
19	quest-2abd9ee775a27361	36	2026-06-18 07:07:13.60353
21	quest-43591853569cdf1e	38	2026-06-18 07:11:47.482323
22	quest-56f37912118a5ce8	39	2026-06-18 07:17:39.350919
23	quest-db39299c4c9bcef0	40	2026-06-18 07:18:41.844734
24	quest-581eb51d0e9beae2	41	2026-06-18 07:20:35.693305
25	quest-82dd100e9a49933f	42	2026-06-18 07:25:18.211871
26	quest-945bccac0e1f562c	43	2026-06-18 07:27:30.152633
28	quest-69085d87018356ed	45	2026-06-18 07:30:31.075958
30	quest-d0bd40e11ac83abb	48	2026-06-18 07:39:23.249827
32	quest-2be911325aa96e09	52	2026-06-18 08:14:57.780828
33	quest-458151fb13775fb8	53	2026-06-18 08:19:11.927162
34	quest-d1ff2d073d0946a5	55	2026-06-18 08:41:55.285123
35	quest-e2acfdd55cc905b5	56	2026-06-18 08:48:58.954905
36	quest-e2acfdd55cc905b5	57	2026-06-18 08:51:15.207193
37	quest-e2acfdd55cc905b5	58	2026-06-18 08:59:24.34266
38	quest-565300dc65df1dea	60	2026-06-18 09:23:18.517778
39	quest-f003bc89b0fab413	61	2026-06-18 09:24:18.425525
40	quest-0010dc4164e3b93a	62	2026-06-18 09:45:15.110963
41	quest-cee9764bfa535831	64	2026-06-18 09:59:25.146625
42	quest-dcab40819162e2b6	65	2026-06-18 10:10:22.502427
43	quest-334ca5c72d8067c0	66	2026-06-18 10:41:19.212193
44	quest-44710e23103e07c9	67	2026-06-18 10:44:07.6936
45	quest-40d82d647bfe2be1	68	2026-06-18 10:46:51.171982
46	quest-dcab40819162e2b6	69	2026-06-18 10:56:55.066472
47	quest-996da99215e74b0c	70	2026-06-18 11:27:06.461744
48	quest-ce6f205d2aa09181	71	2026-06-18 11:27:35.474794
49	quest-565300dc65df1dea	87	2026-06-18 13:18:13.175388
50	quest-d1ff2d073d0946a5	88	2026-06-18 13:24:42.314764
51	quest-0fe368e15274ca5d	89	2026-06-18 13:31:09.436948
52	quest-d77d641aadf22dcf	91	2026-06-18 13:34:57.927494
53	quest-76f4c852354f1018	92	2026-06-18 13:38:23.072903
54	quest-8d10f413d9d3fbe6	93	2026-06-18 13:40:24.18385
55	quest-05d113267d475d32	94	2026-06-18 13:48:53.706043
56	quest-f424c0a43ac11537	95	2026-06-18 13:48:58.106773
57	quest-ea3ddb20b8134840	96	2026-06-18 14:05:43.708418
58	quest-ea3ddb20b8134840	97	2026-06-18 14:16:01.218884
59	quest-871913dafd5ec84e	98	2026-06-18 14:20:46.856011
60	quest-8ec4c33a58571329	99	2026-06-18 14:22:50.94308
61	quest-2a88469f39ea8692	100	2026-06-18 14:24:52.988127
62	quest-3240059b94ea3e00	101	2026-06-18 14:33:32.083624
63	quest-7455e685bc8ab485	102	2026-06-18 14:35:04.451958
64	quest-d6b576b204cf65ba	104	2026-06-18 14:52:08.561987
65	quest-4765cb79955b0dea	105	2026-06-18 14:59:24.494809
66	quest-aa9de3c18964fbb7	106	2026-06-18 15:38:49.793932
67	quest-fdb79a8ccd294c07	107	2026-06-18 15:50:42.131758
68	quest-8ec4c33a58571329	109	2026-06-18 16:24:25.197165
69	quest-fdb79a8ccd294c07	115	2026-06-18 16:50:20.039975
70	quest-aa9de3c18964fbb7	116	2026-06-18 16:51:01.073252
71	quest-fdb79a8ccd294c07	117	2026-06-18 17:04:52.888615
72	quest-a1c284b19792cf09	119	2026-06-18 17:47:00.318744
73	quest-df25a356d916ca48	121	2026-06-18 19:17:06.069944
74	quest-fdb79a8ccd294c07	123	2026-06-19 06:21:43.673024
75	quest-74ca9cd8f26284ed	126	2026-06-19 06:25:41.682828
76	quest-ba4820493933d314	136	2026-06-19 08:35:40.433751
77	quest-ba4820493933d314	137	2026-06-19 08:49:29.75394
78	quest-a096f98deb5fda12	138	2026-06-19 08:50:49.77208
79	quest-74ca9cd8f26284ed	139	2026-06-19 08:51:30.136738
80	quest-ea3ddb20b8134840	140	2026-06-19 09:03:07.558331
81	quest-ea3ddb20b8134840	141	2026-06-19 09:11:27.494738
82	quest-00ae0dc918b16c6f	154	2026-06-19 14:46:27.855167
\.


ALTER TABLE public.quest_transfers ENABLE TRIGGER ALL;

--
-- Data for Name: releases; Type: TABLE DATA; Schema: public; Owner: -
--

ALTER TABLE public.releases DISABLE TRIGGER ALL;

COPY public.releases (id, reference, origin_id, notes, status, created_by, completed_at, created_at) FROM stdin;
1	WYD-2026-001	10	\N	draft	9	\N	2026-06-21 10:19:34.900514
3	WYD-2026-002	10	\N	draft	9	\N	2026-06-21 11:55:58.406664
4	WYD-2026-003	3	\N	draft	9	\N	2026-06-22 09:46:32.804936
5	WYD-2026-004	2	Notatka do wydania WYD-2026-004	draft	9	\N	2026-06-22 10:12:33.719578
6	WYD-2026-005	2	Notatka do wydania WYD-2026-005	draft	9	\N	2026-06-22 10:13:19.019784
\.


ALTER TABLE public.releases ENABLE TRIGGER ALL;

--
-- Data for Name: release_assets; Type: TABLE DATA; Schema: public; Owner: -
--

ALTER TABLE public.release_assets DISABLE TRIGGER ALL;

COPY public.release_assets (id, release_id, item_id, pyr_code, item_serial, category_name, origin_label, location_name) FROM stdin;
1	1	2	PYR-Ekr1	Brak	Ekran projekcyjny	innowacja	Magazyn Techniczny
2	1	250	PYR-GIM1	\N	Glosniki i mikrofon	innowacja	Magazyn Techniczny
3	1	305	PYR-GIM6	\N	Glosniki i mikrofon	innowacja	Magazyn Techniczny
4	1	1	PYR-PRO1	E02GA8202603A	Projektor	innowacja	Magazyn Techniczny
70	3	253	PYR-GIM4	\N	Glosniki i mikrofon	innowacja	Magazyn Techniczny
71	4	40	PYR-LAP34	\N	Laptop	netland	Magazyn Techniczny
72	4	7	PYR-LAP1	LAP11540226500	Laptop	netland	Magazyn Techniczny
73	4	41	PYR-LAP35	\N	Laptop	netland	Magazyn Techniczny
74	4	50	PYR-LAP44	\N	Laptop	netland	Magazyn Techniczny
75	4	8	PYR-LAP2	\N	Laptop	netland	Magazyn Techniczny
76	4	44	PYR-LAP38	\N	Laptop	netland	Magazyn Techniczny
77	4	47	PYR-LAP41	\N	Laptop	netland	Magazyn Techniczny
78	4	48	PYR-LAP42	\N	Laptop	netland	Magazyn Techniczny
79	4	42	PYR-LAP36	\N	Laptop	netland	Magazyn Techniczny
80	4	23	PYR-LAP17	\N	Laptop	netland	Magazyn Techniczny
81	4	20	PYR-LAP14	\N	Laptop	netland	Magazyn Techniczny
82	4	46	PYR-LAP40	\N	Laptop	netland	Magazyn Techniczny
83	4	51	PYR-LAP45	\N	Laptop	netland	Magazyn Techniczny
84	4	36	PYR-LAP30	\N	Laptop	netland	Magazyn Techniczny
85	4	21	PYR-LAP15	\N	Laptop	netland	Magazyn Techniczny
86	4	49	PYR-LAP43	\N	Laptop	netland	Magazyn Techniczny
87	4	39	PYR-LAP33	\N	Laptop	netland	Magazyn Techniczny
88	4	64	PYR-LAP58	\N	Laptop	netland	Magazyn Techniczny
89	4	56	PYR-LAP50	\N	Laptop	netland	Magazyn Techniczny
90	4	22	PYR-LAP16	\N	Laptop	netland	Magazyn Techniczny
91	4	24	PYR-LAP18	\N	Laptop	netland	Magazyn Techniczny
92	4	25	PYR-LAP19	\N	Laptop	netland	Magazyn Techniczny
93	4	28	PYR-LAP22	\N	Laptop	netland	Magazyn Techniczny
94	4	29	PYR-LAP23	\N	Laptop	netland	Magazyn Techniczny
95	4	43	PYR-LAP37	\N	Laptop	netland	Magazyn Techniczny
96	4	63	PYR-LAP57	\N	Laptop	netland	Magazyn Techniczny
97	4	66	PYR-LAP60	\N	Laptop	netland	Magazyn Techniczny
98	4	55	PYR-LAP49	\N	Laptop	netland	Magazyn Techniczny
99	4	34	PYR-LAP28	\N	Laptop	netland	Magazyn Techniczny
100	4	45	PYR-LAP39	\N	Laptop	netland	Magazyn Techniczny
101	4	52	PYR-LAP46	\N	Laptop	netland	Magazyn Techniczny
102	4	62	PYR-LAP56	\N	Laptop	netland	Magazyn Techniczny
103	4	65	PYR-LAP59	\N	Laptop	netland	Magazyn Techniczny
104	4	67	PYR-LAP61	\N	Laptop	netland	Magazyn Techniczny
105	4	68	PYR-LAP62	\N	Laptop	netland	Magazyn Techniczny
106	4	54	PYR-LAP48	\N	Laptop	netland	Magazyn Techniczny
107	4	57	PYR-LAP51	\N	Laptop	netland	Magazyn Techniczny
108	4	58	PYR-LAP52	\N	Laptop	netland	Magazyn Techniczny
109	4	59	PYR-LAP53	\N	Laptop	netland	Magazyn Techniczny
110	4	60	PYR-LAP54	\N	Laptop	netland	Magazyn Techniczny
111	4	61	PYR-LAP55	\N	Laptop	netland	Magazyn Techniczny
112	4	19	PYR-LAP13	\N	Laptop	netland	Magazyn Techniczny
113	4	17	PYR-LAP11	\N	Laptop	netland	Magazyn Techniczny
114	4	18	PYR-LAP12	\N	Laptop	netland	Magazyn Techniczny
115	4	30	PYR-LAP24	\N	Laptop	netland	Magazyn Techniczny
116	4	14	PYR-LAP8	\N	Laptop	netland	Magazyn Techniczny
117	4	15	PYR-LAP9	\N	Laptop	netland	Magazyn Techniczny
118	4	69	PYR-LAP63	\N	Laptop	netland	Magazyn Techniczny
119	4	9	PYR-LAP3	\N	Laptop	netland	Magazyn Techniczny
120	4	16	PYR-LAP10	\N	Laptop	netland	Magazyn Techniczny
121	4	10	PYR-LAP4	\N	Laptop	netland	Magazyn Techniczny
122	4	11	PYR-LAP5	\N	Laptop	netland	Magazyn Techniczny
123	4	12	PYR-LAP6	\N	Laptop	netland	Magazyn Techniczny
124	4	13	PYR-LAP7	\N	Laptop	netland	Magazyn Techniczny
125	4	26	PYR-LAP20	\N	Laptop	netland	Magazyn Techniczny
126	4	27	PYR-LAP21	\N	Laptop	netland	Magazyn Techniczny
127	4	70	PYR-LAP64	\N	Laptop	netland	Magazyn Techniczny
128	4	31	PYR-LAP25	\N	Laptop	netland	Magazyn Techniczny
129	4	71	PYR-LAP65	\N	Laptop	netland	Magazyn Techniczny
130	4	32	PYR-LAP26	\N	Laptop	netland	Magazyn Techniczny
131	4	33	PYR-LAP27	\N	Laptop	netland	Magazyn Techniczny
132	4	35	PYR-LAP29	\N	Laptop	netland	Magazyn Techniczny
133	4	37	PYR-LAP31	\N	Laptop	netland	Magazyn Techniczny
134	4	38	PYR-LAP32	\N	Laptop	netland	Magazyn Techniczny
135	4	53	PYR-LAP47	\N	Laptop	netland	Magazyn Techniczny
136	5	3	PYR-PA41	VNC4C33551	Drukarka A4	probis	Magazyn Techniczny
137	5	6	PYR-PA44	VNC4D69420	Drukarka A4	probis	Magazyn Techniczny
138	5	74	PYR-PA45	0	Drukarka A4	probis	Magazyn Techniczny
139	5	87	PYR-PA416	011	Drukarka A4	probis	Magazyn Techniczny
140	5	77	PYR-PA46	01	Drukarka A4	probis	Magazyn Techniczny
141	5	84	PYR-PA413	08	Drukarka A4	probis	Magazyn Techniczny
142	5	5	PYR-PA43	VNC4B91585	Drukarka A4	probis	Magazyn Techniczny
143	5	97	PYR-LAP66	\N	Laptop	probis	Magazyn Techniczny
144	5	94	\N	1011	Drukarka A4	probis	Magazyn Techniczny
145	5	95	\N	001	Drukarka A4	probis	Magazyn Techniczny
146	5	85	PYR-PA414	09	Drukarka A4	probis	Magazyn Techniczny
147	5	82	PYR-PA411	06	Drukarka A4	probis	Magazyn Techniczny
148	5	89	PYR-PA418	013	Drukarka A4	probis	Magazyn Techniczny
149	5	98	PYR-LAP67	\N	Laptop	probis	Magazyn Techniczny
150	5	99	PYR-LAP68	\N	Laptop	probis	Magazyn Techniczny
151	5	100	PYR-LAP69	\N	Laptop	probis	Magazyn Techniczny
152	5	106	PYR-LAP75	\N	Laptop	probis	Magazyn Techniczny
153	5	101	PYR-LAP70	\N	Laptop	probis	Magazyn Techniczny
154	5	78	PYR-PA47	02	Drukarka A4	probis	Magazyn Techniczny
155	5	104	PYR-LAP73	\N	Laptop	probis	Magazyn Techniczny
156	5	105	PYR-LAP74	\N	Laptop	probis	Magazyn Techniczny
157	5	107	PYR-LAP76	\N	Laptop	probis	Magazyn Techniczny
158	5	88	PYR-PA417	012	Drukarka A4	probis	Magazyn Techniczny
159	5	79	PYR-PA48	03	Drukarka A4	probis	Magazyn Techniczny
160	5	86	PYR-PA415	010	Drukarka A4	probis	Magazyn Techniczny
161	5	90	PYR-PA419	014	Drukarka A4	probis	Magazyn Techniczny
162	5	83	PYR-PA412	07	Drukarka A4	probis	Magazyn Techniczny
163	5	103	PYR-LAP72	\N	Laptop	probis	Magazyn Techniczny
164	5	102	PYR-LAP71	\N	Laptop	probis	Magazyn Techniczny
165	5	92	PYR-PA421	016	Drukarka A4	probis	Magazyn Techniczny
166	5	299	PYR-LAP87	\N	Laptop	probis	Magazyn Techniczny
167	5	300	PYR-LAP88	\N	Laptop	probis	Magazyn Techniczny
168	5	93	PYR-PA422	017	Drukarka A4	probis	Magazyn Techniczny
169	5	298	PYR-LAP86	\N	Laptop	probis	Magazyn Techniczny
170	5	91	PYR-PA420	015	Drukarka A4	probis	Magazyn Techniczny
171	5	301	PYR-LAP89	\N	Laptop	probis	Magazyn Techniczny
172	5	302	PYR-LAP90	\N	Laptop	probis	Magazyn Techniczny
173	5	304	PYR-LAP92	\N	Laptop	probis	Magazyn Techniczny
174	5	303	PYR-LAP91	\N	Laptop	probis	Magazyn Techniczny
175	5	110	PYR-LAP79	\N	Laptop	probis	Magazyn Techniczny
176	5	111	PYR-LAP80	\N	Laptop	probis	Magazyn Techniczny
177	5	108	PYR-LAP77	\N	Laptop	probis	Magazyn Techniczny
178	5	114	PYR-LAP83	\N	Laptop	probis	Magazyn Techniczny
179	5	115	PYR-LAP84	\N	Laptop	probis	Magazyn Techniczny
180	5	109	PYR-LAP78	\N	Laptop	probis	Magazyn Techniczny
181	5	112	PYR-LAP81	\N	Laptop	probis	Magazyn Techniczny
182	5	113	PYR-LAP82	\N	Laptop	probis	Magazyn Techniczny
183	5	117	\N	0123	Drukarka A4	probis	Magazyn Techniczny
184	5	296	PYR-SPB2	\N	Głośniki (Duże)	probis	Magazyn Techniczny
185	5	81	PYR-PA410	05	Drukarka A4	probis	Magazyn Techniczny
186	5	116	PYR-LAP85	\N	Laptop	probis	Magazyn Techniczny
187	5	80	PYR-PA49	04	Drukarka A4	probis	Magazyn Techniczny
188	5	119	PYR-PA31	\N	Drukarka A3	probis	Magazyn Techniczny
189	6	166	PYR-TVX21	\N	TV	probis	Magazyn Techniczny
190	6	169	PYR-TVX24	\N	TV	probis	Magazyn Techniczny
191	6	170	PYR-TVX25	\N	TV	probis	Magazyn Techniczny
192	6	171	PYR-TVX26	\N	TV	probis	Magazyn Techniczny
193	6	172	PYR-TVX27	\N	TV	probis	Magazyn Techniczny
194	6	173	PYR-TVX28	\N	TV	probis	Magazyn Techniczny
195	6	174	PYR-TVX29	\N	TV	probis	Magazyn Techniczny
196	6	175	PYR-TVX30	\N	TV	probis	Magazyn Techniczny
197	6	176	PYR-TVX31	\N	TV	probis	Magazyn Techniczny
198	6	177	PYR-TVX32	\N	TV	probis	Magazyn Techniczny
199	6	178	PYR-TVX33	\N	TV	probis	Magazyn Techniczny
200	6	195	PYR-T751	\N	TV 75	probis	Magazyn Techniczny
201	6	196	PYR-T752	\N	TV 75	probis	Magazyn Techniczny
\.


ALTER TABLE public.release_assets ENABLE TRIGGER ALL;

--
-- Data for Name: release_stocks; Type: TABLE DATA; Schema: public; Owner: -
--

ALTER TABLE public.release_stocks DISABLE TRIGGER ALL;

COPY public.release_stocks (id, release_id, stock_id, item_category_id, category_name, quantity, origin_label, location_name) FROM stdin;
\.


ALTER TABLE public.release_stocks ENABLE TRIGGER ALL;

--
-- Data for Name: schedules; Type: TABLE DATA; Schema: public; Owner: -
--

ALTER TABLE public.schedules DISABLE TRIGGER ALL;

COPY public.schedules (id, name, event_start, event_end, festival_start, festival_end, status, created_at, version) FROM stdin;
10	Pyrkon 2026	2026-06-16	2026-06-22	2026-06-19 10:00:00	2026-06-21 21:37:00	active	2026-05-06 21:40:40.898517	126
\.


ALTER TABLE public.schedules ENABLE TRIGGER ALL;

--
-- Data for Name: schedule_slots; Type: TABLE DATA; Schema: public; Owner: -
--

ALTER TABLE public.schedule_slots DISABLE TRIGGER ALL;

COPY public.schedule_slots (id, schedule_id, slot_type, start_time, end_time, credit_hours, capacity, label) FROM stdin;
293	10	montage	2026-06-16 11:00:00	2026-06-16 13:00:00	2.0	4	Montaż - Wt 11:00 - 13:00
229	10	festival	2026-06-19 10:00:00	2026-06-19 11:00:00	1.0	2	Festiwal 19.06 12:00
230	10	festival	2026-06-19 11:00:00	2026-06-19 12:00:00	1.0	2	Festiwal 19.06 13:00
231	10	festival	2026-06-19 12:00:00	2026-06-19 13:00:00	1.0	2	Festiwal 19.06 14:00
232	10	festival	2026-06-19 13:00:00	2026-06-19 14:00:00	1.0	2	Festiwal 19.06 15:00
233	10	festival	2026-06-19 14:00:00	2026-06-19 15:00:00	1.0	2	Festiwal 19.06 16:00
234	10	festival	2026-06-19 15:00:00	2026-06-19 16:00:00	1.0	2	Festiwal 19.06 17:00
235	10	festival	2026-06-19 16:00:00	2026-06-19 17:00:00	1.0	2	Festiwal 19.06 18:00
236	10	festival	2026-06-19 17:00:00	2026-06-19 18:00:00	1.0	2	Festiwal 19.06 19:00
237	10	festival	2026-06-19 18:00:00	2026-06-19 19:00:00	1.0	2	Festiwal 19.06 20:00
238	10	festival	2026-06-19 19:00:00	2026-06-19 20:00:00	1.0	2	Festiwal 19.06 21:00
239	10	festival	2026-06-19 20:00:00	2026-06-19 21:00:00	1.0	2	Festiwal 19.06 22:00
240	10	festival	2026-06-19 21:00:00	2026-06-19 22:00:00	1.0	2	Festiwal 19.06 23:00
241	10	festival	2026-06-19 22:00:00	2026-06-19 23:00:00	1.0	2	Festiwal 20.06 00:00
242	10	festival	2026-06-19 23:00:00	2026-06-20 00:00:00	1.0	2	Festiwal 20.06 01:00
243	10	festival	2026-06-20 00:00:00	2026-06-20 01:00:00	1.0	2	Festiwal 20.06 02:00
244	10	festival	2026-06-20 01:00:00	2026-06-20 02:00:00	1.0	2	Festiwal 20.06 03:00
245	10	festival	2026-06-20 02:00:00	2026-06-20 03:00:00	1.0	2	Festiwal 20.06 04:00
246	10	festival	2026-06-20 03:00:00	2026-06-20 04:00:00	1.0	2	Festiwal 20.06 05:00
247	10	festival	2026-06-20 04:00:00	2026-06-20 05:00:00	1.0	2	Festiwal 20.06 06:00
248	10	festival	2026-06-20 05:00:00	2026-06-20 06:00:00	1.0	2	Festiwal 20.06 07:00
249	10	festival	2026-06-20 06:00:00	2026-06-20 07:00:00	1.0	2	Festiwal 20.06 08:00
250	10	festival	2026-06-20 07:00:00	2026-06-20 08:00:00	1.0	2	Festiwal 20.06 09:00
251	10	festival	2026-06-20 08:00:00	2026-06-20 09:00:00	1.0	2	Festiwal 20.06 10:00
252	10	festival	2026-06-20 09:00:00	2026-06-20 10:00:00	1.0	2	Festiwal 20.06 11:00
253	10	festival	2026-06-20 10:00:00	2026-06-20 11:00:00	1.0	2	Festiwal 20.06 12:00
254	10	festival	2026-06-20 11:00:00	2026-06-20 12:00:00	1.0	2	Festiwal 20.06 13:00
255	10	festival	2026-06-20 12:00:00	2026-06-20 13:00:00	1.0	2	Festiwal 20.06 14:00
256	10	festival	2026-06-20 13:00:00	2026-06-20 14:00:00	1.0	2	Festiwal 20.06 15:00
257	10	festival	2026-06-20 14:00:00	2026-06-20 15:00:00	1.0	2	Festiwal 20.06 16:00
258	10	festival	2026-06-20 15:00:00	2026-06-20 16:00:00	1.0	2	Festiwal 20.06 17:00
259	10	festival	2026-06-20 16:00:00	2026-06-20 17:00:00	1.0	2	Festiwal 20.06 18:00
260	10	festival	2026-06-20 17:00:00	2026-06-20 18:00:00	1.0	2	Festiwal 20.06 19:00
261	10	festival	2026-06-20 18:00:00	2026-06-20 19:00:00	1.0	2	Festiwal 20.06 20:00
262	10	festival	2026-06-20 19:00:00	2026-06-20 20:00:00	1.0	2	Festiwal 20.06 21:00
263	10	festival	2026-06-20 20:00:00	2026-06-20 21:00:00	1.0	2	Festiwal 20.06 22:00
264	10	festival	2026-06-20 21:00:00	2026-06-20 22:00:00	1.0	2	Festiwal 20.06 23:00
265	10	festival	2026-06-20 22:00:00	2026-06-20 23:00:00	1.0	2	Festiwal 21.06 00:00
266	10	festival	2026-06-20 23:00:00	2026-06-21 00:00:00	1.0	2	Festiwal 21.06 01:00
267	10	festival	2026-06-21 00:00:00	2026-06-21 01:00:00	1.0	2	Festiwal 21.06 02:00
268	10	festival	2026-06-21 01:00:00	2026-06-21 02:00:00	1.0	2	Festiwal 21.06 03:00
269	10	festival	2026-06-21 02:00:00	2026-06-21 03:00:00	1.0	2	Festiwal 21.06 04:00
270	10	festival	2026-06-21 03:00:00	2026-06-21 04:00:00	1.0	2	Festiwal 21.06 05:00
271	10	festival	2026-06-21 04:00:00	2026-06-21 05:00:00	1.0	2	Festiwal 21.06 06:00
272	10	festival	2026-06-21 05:00:00	2026-06-21 06:00:00	1.0	2	Festiwal 21.06 07:00
273	10	festival	2026-06-21 06:00:00	2026-06-21 07:00:00	1.0	2	Festiwal 21.06 08:00
274	10	festival	2026-06-21 07:00:00	2026-06-21 08:00:00	1.0	2	Festiwal 21.06 09:00
275	10	festival	2026-06-21 08:00:00	2026-06-21 09:00:00	1.0	2	Festiwal 21.06 10:00
276	10	festival	2026-06-21 09:00:00	2026-06-21 10:00:00	1.0	2	Festiwal 21.06 11:00
277	10	festival	2026-06-21 10:00:00	2026-06-21 11:00:00	1.0	2	Festiwal 21.06 12:00
278	10	festival	2026-06-21 11:00:00	2026-06-21 12:00:00	1.0	2	Festiwal 21.06 13:00
279	10	festival	2026-06-21 12:00:00	2026-06-21 13:00:00	1.0	2	Festiwal 21.06 14:00
280	10	festival	2026-06-21 13:00:00	2026-06-21 14:00:00	1.0	2	Festiwal 21.06 15:00
281	10	festival	2026-06-21 14:00:00	2026-06-21 15:00:00	1.0	2	Festiwal 21.06 16:00
282	10	festival	2026-06-21 15:00:00	2026-06-21 16:00:00	1.0	2	Festiwal 21.06 17:00
283	10	festival	2026-06-21 16:00:00	2026-06-21 17:00:00	1.0	2	Festiwal 21.06 18:00
284	10	festival	2026-06-21 17:00:00	2026-06-21 18:00:00	1.0	2	Festiwal 21.06 19:00
285	10	festival	2026-06-21 18:00:00	2026-06-21 19:00:00	1.0	2	Festiwal 21.06 20:00
286	10	festival	2026-06-21 19:00:00	2026-06-21 20:00:00	1.0	2	Festiwal 21.06 21:00
287	10	festival	2026-06-21 20:00:00	2026-06-21 21:00:00	1.0	2	Festiwal 21.06 22:00
330	10	montage	2026-06-16 15:00:00	2026-06-16 17:00:00	2.0	2	 Montaż - Wt 15:00-17:00
289	10	festival	2026-06-19 09:00:00	2026-06-19 10:00:00	1.0	0	Festiwal pt., 19.06
290	10	festival	2026-06-19 08:00:00	2026-06-19 09:00:00	1.0	0	Festiwal pt., 19.06
335	10	montage	2026-06-18 13:00:00	2026-06-18 14:00:00	1.0	0	 Montaż - Czw 13:00-14:00
333	10	montage	2026-06-18 17:00:00	2026-06-18 18:00:00	1.0	5	Montaż - Czw 17:00-18:00
337	10	montage	2026-06-18 19:00:00	2026-06-18 20:00:00	1.0	3	Montaż - Czw 19:00-20:00
332	10	demontage	2026-06-22 11:00:00	2026-06-22 13:00:00	2.0	4	Demon pon., 22.06
297	10	montage	2026-06-16 13:00:00	2026-06-16 15:00:00	2.0	2	Montaż - Wt 13:00-15:00
301	10	montage	2026-06-17 11:00:00	2026-06-17 12:00:00	1.0	2	Montaż - Śr 11:00-12:00
303	10	montage	2026-06-17 12:00:00	2026-06-17 14:00:00	2.0	7	Montaż - Śr 12:00-14:00
339	10	montage	2026-06-17 14:00:00	2026-06-17 15:00:00	1.0	6	Montaż - Śr 14:00-15:00
305	10	montage	2026-06-17 15:00:00	2026-06-17 16:00:00	1.0	2	Montaż - Śr 15:00-16:00
338	10	montage	2026-06-17 16:00:00	2026-06-17 17:00:00	1.0	6	 Montaż - Śr 16:00-17:00
307	10	montage	2026-06-17 17:00:00	2026-06-17 19:00:00	2.0	3	Montaż - Śr 17:00-19:00
313	10	montage	2026-06-18 11:00:00	2026-06-18 12:00:00	1.0	7	Montaż - Czw 11:00-12:00
314	10	montage	2026-06-18 12:00:00	2026-06-18 13:00:00	1.0	2	Montaż - Czw 12:00-13:00
318	10	montage	2026-06-18 14:00:00	2026-06-18 15:00:00	1.0	6	Montaż - Czw 14:00-15:00
336	10	montage	2026-06-18 15:00:00	2026-06-18 16:00:00	1.0	7	Montaż - Czw 15:00-16:00 
319	10	montage	2026-06-18 16:00:00	2026-06-18 17:00:00	1.0	10	Montaż - Czw 16:00-17:00
321	10	montage	2026-06-18 18:00:00	2026-06-18 19:00:00	1.0	4	Montaż - Czw 18:00-19:00
334	10	demontage	2026-06-22 13:00:00	2026-06-22 15:00:00	2.0	2	Demontaż pon., 22.06
\.


ALTER TABLE public.schedule_slots ENABLE TRIGGER ALL;

--
-- Data for Name: schedule_volunteers; Type: TABLE DATA; Schema: public; Owner: -
--

ALTER TABLE public.schedule_volunteers DISABLE TRIGGER ALL;

COPY public.schedule_volunteers (id, schedule_id, user_id, nickname, city, target_hours, available_from, available_to, notes, assigned_hours, discord_confirmed) FROM stdin;
207	10	66	user66	Lublin	14	2026-06-16 06:00:00	2026-06-22 21:00:00	\N	14.0	discord_user66
200	10	65	user65	Poznań	14	2026-06-19 09:30:00	2026-06-21 14:00:00	\N	14.0	discord_user65
202	10	69	user69	Kraków	14	2026-06-19 04:00:00	2026-06-21 17:00:00	\N	14.0	discord_user69
205	10	70	user70	Łódź	14	2026-06-19 10:00:00	2026-06-21 21:59:00	\N	14.0	discord_user70
189	10	13	user13	Katowice	14	2026-06-16 08:00:00	2026-06-22 18:00:00	\N	14.0	discord_user13
190	10	71	user71	Poznań	14	2026-06-16 12:00:00	2026-06-22 12:00:00	\N	14.0	discord_user71
188	10	108	user108	Bydgoszcz	14	2026-06-18 11:00:00	2026-06-21 15:00:00	\N	14.0	discord_user108
206	10	105	user105	Szczecin	14	2026-06-19 09:00:00	2026-06-21 15:00:00	\N	14.0	discord_user105
195	10	\N	wolontariusz195	Łódź	14	2026-06-19 09:00:00	2026-06-21 13:00:00	\N	14.0	\N
201	10	\N	wolontariusz201	Warszawa	14	2026-06-17 08:00:00	2026-06-21 16:00:00	\N	0.0	\N
203	10	67	user67	Wrocław	14	2026-06-15 22:00:00	2026-06-21 22:00:00	\N	14.0	discord_user67
191	10	\N	wolontariusz191	Warszawa	14	2026-06-16 08:00:00	2026-06-21 20:00:00	\N	14.0	discord_user191
187	10	\N	wolontariusz187	Lublin	14	2026-06-18 12:00:00	2026-06-21 15:00:00	\N	14.0	\N
186	10	\N	wolontariusz186	Szczecin	14	2026-06-19 07:15:00	2026-06-21 14:00:00	\N	0.0	\N
196	10	68	user68	Szczecin	14	2026-06-15 22:00:00	2026-06-22 21:59:00	\N	14.0	discord_user68
193	10	72	user72	Wrocław	14	2026-06-16 10:00:00	2026-06-22 20:00:00	\N	14.0	discord_user72
204	10	106	user106	Gdańsk	14	2026-06-16 12:00:00	2026-06-22 08:00:00	\N	14.0	discord_user106
198	10	109	user109	Bydgoszcz	14	2026-06-16 12:00:00	2026-06-22 17:00:00	\N	14.0	discord_user109
197	10	112	user112	Lublin	14	2026-06-16 09:00:00	2026-06-21 15:00:00	\N	14.0	discord_user112
192	10	114	user114	Kraków	14	2026-06-17 22:00:00	2026-06-22 07:00:00	\N	14.0	discord_user114
194	10	110	user110	Gdańsk	14	2026-06-15 22:00:00	2026-06-21 21:00:00	\N	14.0	discord_user110
199	10	111	user111	Katowice	14	2026-06-16 14:00:00	2026-06-22 14:00:00	\N	14.0	discord_user111
185	10	\N	wolontariusz185	Łódź	14	2026-06-18 12:00:00	2026-06-21 15:00:00	\N	14.0	discord_user185
\.


ALTER TABLE public.schedule_volunteers ENABLE TRIGGER ALL;

--
-- Data for Name: schedule_assignments; Type: TABLE DATA; Schema: public; Owner: -
--

ALTER TABLE public.schedule_assignments DISABLE TRIGGER ALL;

COPY public.schedule_assignments (id, slot_id, volunteer_id) FROM stdin;
31770	293	189
31771	293	194
31772	293	202
31773	297	189
31774	297	194
31775	297	202
31776	297	197
31777	330	189
31778	330	194
31779	330	202
31780	301	190
31781	301	197
31782	301	199
31783	301	204
31784	301	191
31785	301	203
31786	303	190
31787	303	197
31788	303	199
31789	303	204
31790	303	203
31791	303	191
31792	339	190
31793	339	197
31794	339	199
31795	339	204
31796	339	203
31797	339	191
31798	305	190
31799	305	197
31800	305	199
31801	305	204
31802	305	203
31803	305	191
31804	338	190
31805	338	197
31806	338	199
31807	338	204
31808	338	203
31809	338	191
31810	338	196
31811	307	196
31812	307	189
31813	313	188
31814	313	192
31815	313	198
31816	313	189
31817	313	207
31818	313	193
31819	314	188
31820	314	192
31821	314	198
31822	314	189
31823	314	207
31824	314	193
31825	335	188
31826	335	192
31827	335	198
31828	335	189
31829	335	207
31830	335	193
31831	318	188
31832	318	192
31833	318	198
31834	318	189
31835	318	207
31836	318	193
31837	336	188
31838	336	192
31839	336	198
31840	336	189
31841	336	207
31842	336	193
31843	319	193
31844	319	202
31845	319	189
31846	319	192
31847	319	205
31848	319	198
31849	319	188
31850	319	207
31851	333	196
31852	333	202
31853	333	205
31854	321	196
31855	321	202
31856	321	205
31857	337	196
31858	337	202
31859	337	205
31860	290	199
31861	290	197
31862	290	188
31863	289	199
31864	289	197
31865	289	188
31866	229	199
31867	229	197
31868	229	198
31869	230	199
31870	230	197
31871	230	198
31872	231	198
31873	231	193
31874	231	206
31875	232	198
31876	232	193
31877	232	206
31878	233	206
31879	233	195
31880	233	200
31881	233	187
31882	233	185
31883	234	206
31884	234	195
31885	234	200
31886	234	187
31887	234	185
31888	235	195
31889	235	200
31890	235	187
31891	235	185
31892	236	195
31893	236	200
31894	236	204
31895	237	204
31896	237	203
31897	238	204
31898	238	203
31899	239	204
31900	239	191
31901	240	196
31902	240	191
31903	241	196
31904	241	191
31905	242	205
31906	242	193
31907	243	205
31908	243	193
31909	244	185
31910	244	187
31911	245	185
31912	245	187
31913	246	185
31914	246	187
31915	247	185
31916	247	187
31917	248	185
31918	248	187
31919	249	185
31920	249	187
31921	250	190
31922	250	200
31923	251	190
31924	251	200
31925	252	190
31926	252	200
31927	253	195
31928	253	198
31929	253	206
31930	254	195
31931	254	198
31932	254	206
31933	255	195
31934	255	198
31935	255	206
31936	255	193
31937	256	195
31938	256	198
31939	256	206
31940	256	193
31941	257	196
31942	257	188
31943	257	205
31944	258	196
31945	258	188
31946	258	205
31947	259	203
31948	259	188
31949	260	203
31950	260	187
31951	260	185
31952	261	199
31953	261	192
31954	262	199
31955	262	192
31956	263	199
31957	263	192
31958	264	199
31959	264	192
31960	265	195
31961	265	200
31962	266	195
31963	266	200
31964	267	195
31965	267	200
31966	268	195
31967	268	200
31968	269	206
31969	269	194
31970	270	206
31971	270	194
31972	271	206
31973	271	194
31974	272	206
31975	272	194
31976	273	194
31977	273	205
31978	274	194
31979	274	205
31980	275	194
31981	275	205
31982	276	194
31983	276	205
31984	277	195
31985	277	200
31986	277	185
31987	277	187
31988	278	195
31989	278	200
31990	278	185
31991	278	187
31992	279	188
31993	279	200
31994	279	185
31995	279	187
31996	280	188
31997	280	197
31998	280	185
31999	280	187
32000	281	190
32001	281	188
32002	281	197
32003	281	192
32004	282	190
32005	282	191
32006	282	206
32007	282	192
32008	282	203
32009	283	191
32010	283	206
32011	283	190
32012	283	192
32013	283	203
32014	284	191
32015	284	204
32016	284	207
32017	284	196
32018	284	202
32019	284	205
32020	284	192
32021	285	191
32022	285	204
32023	285	207
32024	285	196
32025	285	202
32026	285	205
32027	286	191
32028	286	204
32029	286	207
32030	286	196
32031	286	202
32032	287	204
32033	287	207
32034	287	196
32035	287	202
32036	332	207
32037	332	193
32038	332	203
32039	332	190
32040	334	207
\.


ALTER TABLE public.schedule_assignments ENABLE TRIGGER ALL;

--
-- Data for Name: schedule_day_windows; Type: TABLE DATA; Schema: public; Owner: -
--

ALTER TABLE public.schedule_day_windows DISABLE TRIGGER ALL;

COPY public.schedule_day_windows (id, schedule_id, date, window_start, window_end) FROM stdin;
1	10	2026-06-16	10:00:00	18:00:00
2	10	2026-06-17	10:00:00	20:00:00
3	10	2026-06-18	08:00:00	22:00:00
\.


ALTER TABLE public.schedule_day_windows ENABLE TRIGGER ALL;

--
-- Data for Name: serialized_transfers; Type: TABLE DATA; Schema: public; Owner: -
--

ALTER TABLE public.serialized_transfers DISABLE TRIGGER ALL;

COPY public.serialized_transfers (id, transfer_id, item_id) FROM stdin;
1	1	2
2	1	1
3	2	3
4	4	7
5	6	72
6	6	73
7	7	119
8	7	40
9	7	41
10	7	81
11	8	121
12	9	122
13	9	4
14	10	6
15	11	48
16	11	47
17	11	8
18	11	9
19	11	10
20	11	11
21	11	12
22	11	14
23	11	13
24	11	16
25	11	15
26	23	120
27	24	184
28	24	186
29	24	108
30	24	109
31	25	189
32	26	87
33	28	144
34	28	42
35	29	83
36	29	202
37	29	201
38	29	203
40	29	205
41	29	206
42	29	110
43	29	111
44	29	112
45	30	194
46	31	148
47	31	142
48	31	146
49	31	165
50	31	147
51	31	164
52	32	190
53	33	218
54	34	231
55	36	137
56	36	234
57	37	185
58	37	226
59	38	185
60	38	226
61	39	84
62	40	44
63	42	238
64	42	237
65	42	236
66	42	235
67	43	180
68	44	53
69	45	46
70	46	229
71	47	45
72	48	36
73	50	34
74	52	38
75	52	37
76	52	53
77	52	35
78	52	33
79	52	32
80	52	30
81	52	31
82	52	29
83	52	28
84	52	193
85	53	239
86	54	121
87	55	204
88	55	63
89	55	65
90	55	66
91	55	67
92	55	43
93	55	69
94	55	68
95	55	64
96	55	39
97	55	70
98	55	71
99	55	240
100	55	241
101	55	242
102	55	93
103	55	5
104	55	183
105	55	244
106	56	80
107	56	213
108	56	101
109	56	107
110	56	105
111	56	104
112	56	222
113	57	155
114	60	97
118	61	126
119	61	127
120	63	250
121	64	149
122	64	150
123	64	157
124	65	52
125	65	34
126	65	62
127	65	45
128	65	91
129	65	88
130	65	251
131	65	252
132	66	82
133	66	89
134	66	98
135	66	99
136	66	100
137	66	106
138	66	243
139	66	181
140	66	259
141	66	260
142	66	261
143	67	116
144	67	115
145	67	114
146	67	254
147	68	262
148	69	138
149	69	139
150	69	145
151	69	166
152	69	152
153	69	151
154	69	153
155	69	154
156	69	255
157	69	256
158	69	257
159	69	258
160	73	51
183	87	175
184	87	176
185	87	177
186	88	267
187	88	266
188	88	232
189	88	265
190	89	228
191	89	22
192	89	24
193	89	25
194	89	26
195	89	27
196	89	216
197	89	78
198	89	178
199	91	55
200	92	268
201	95	54
202	95	57
203	95	171
204	95	172
205	96	173
206	96	174
207	96	169
208	96	170
209	97	85
210	97	23
211	97	56
212	97	59
213	97	58
214	97	61
215	97	60
216	98	270
217	98	269
218	99	125
219	99	123
220	99	131
221	101	21
222	102	192
223	103	113
224	104	49
225	104	90
226	105	191
227	105	86
228	105	271
229	105	272
230	105	273
231	105	274
232	105	275
233	105	182
234	105	276
235	105	277
236	105	278
237	106	253
238	106	19
239	107	20
240	107	214
241	109	279
242	109	280
243	109	281
244	109	282
245	113	18
246	113	17
247	113	103
248	115	284
249	115	285
250	115	286
251	115	287
252	115	288
253	115	289
254	115	283
255	116	290
256	117	136
257	117	134
258	117	132
259	117	133
260	117	130
261	117	128
262	117	129
263	117	124
264	118	111
265	119	190
266	121	141
267	122	2
268	122	1
269	123	291
270	123	292
271	123	293
272	123	294
273	123	295
274	124	50
275	125	110
276	126	110
277	127	296
278	128	220
279	129	297
280	130	217
281	131	233
282	134	79
283	135	80
284	137	299
285	138	298
286	139	300
287	140	140
288	140	143
289	141	303
290	142	23
291	147	121
292	151	199
293	152	143
294	154	302
295	154	304
296	154	301
297	156	102
298	158	78
299	159	80
300	160	305
301	161	43
302	162	297
303	163	250
304	164	23
305	165	23
306	166	304
307	166	302
308	166	301
309	167	253
310	168	29
311	169	14
312	170	84
313	170	137
314	171	238
315	171	236
316	171	237
317	171	235
318	171	44
319	172	51
320	173	213
321	173	155
322	174	19
323	175	287
324	176	234
325	178	103
326	179	141
327	180	169
328	180	173
329	180	174
330	181	170
331	182	17
332	182	18
333	183	216
334	183	228
335	183	80
336	183	26
337	183	24
338	183	25
339	183	22
340	183	27
341	184	178
342	185	303
343	185	60
344	185	61
345	185	58
346	185	56
347	185	59
348	187	140
349	188	289
350	188	284
351	190	57
352	190	54
353	192	199
354	194	220
355	194	90
356	194	49
357	196	192
358	196	21
359	197	82
360	197	89
361	197	98
362	197	99
363	197	100
364	197	106
365	197	243
366	197	259
367	197	260
368	197	114
369	197	115
370	197	116
371	197	254
372	197	181
373	197	261
374	197	121
375	198	123
376	198	125
377	198	131
378	199	193
379	199	33
380	199	38
381	199	35
382	199	31
383	199	30
384	199	32
385	199	28
386	199	53
387	199	37
388	200	233
389	201	85
390	204	128
391	204	129
392	204	124
393	204	132
394	204	133
395	204	130
396	204	134
397	204	295
398	204	294
399	204	293
400	204	292
401	204	291
402	204	136
403	204	286
404	204	283
405	204	288
406	204	214
407	204	20
408	207	229
409	208	16
410	208	9
411	208	46
412	208	180
413	209	279
414	209	280
415	209	281
416	209	282
417	210	86
418	210	102
419	210	275
420	210	276
421	210	271
422	210	273
423	210	278
424	210	274
425	210	277
426	210	272
427	210	182
428	210	191
429	212	194
430	212	55
431	212	268
432	213	239
433	213	218
434	213	206
435	213	205
436	213	203
437	213	202
438	213	201
439	213	113
440	213	112
441	213	83
442	213	50
443	214	232
444	215	34
445	215	45
446	215	52
447	215	62
448	215	88
449	215	166
450	215	255
451	215	256
452	215	257
453	215	258
454	215	91
455	215	251
456	215	252
457	215	138
458	215	139
459	215	145
460	215	151
461	215	152
462	215	153
463	215	154
464	216	185
465	217	290
466	218	262
467	219	231
468	220	299
469	221	270
470	221	269
471	222	110
472	222	300
473	223	101
474	224	36
475	225	15
476	226	217
477	227	3
478	229	267
479	229	266
480	229	265
481	229	242
482	229	241
483	229	240
484	229	93
485	229	71
486	229	70
487	229	69
488	229	68
489	229	67
490	229	66
491	229	64
492	229	39
493	229	5
494	230	105
495	231	298
496	232	107
497	233	120
498	234	104
499	235	8
500	235	47
501	235	48
502	235	10
503	235	11
504	235	12
505	235	13
506	236	184
507	236	189
508	236	108
509	236	87
510	236	146
511	236	147
512	236	148
513	236	164
514	236	296
515	238	285
516	239	172
517	239	171
518	240	81
519	240	119
520	240	40
521	240	41
522	241	183
523	241	63
524	241	65
525	241	244
526	241	204
527	243	190
528	244	79
529	245	226
530	246	165
531	247	126
532	247	127
533	248	97
534	248	175
535	248	176
536	249	177
537	250	186
538	251	109
539	252	7
540	252	6
541	253	43
542	254	42
543	255	144
544	255	149
545	255	150
546	255	157
547	256	142
\.


ALTER TABLE public.serialized_transfers ENABLE TRIGGER ALL;

--
-- Data for Name: service_desk_request_comments; Type: TABLE DATA; Schema: public; Owner: -
--

ALTER TABLE public.service_desk_request_comments DISABLE TRIGGER ALL;

COPY public.service_desk_request_comments (id, request_id, comment, created_at, user_id) FROM stdin;
1	4	Komentarz testowy #1.	2026-06-17 15:30:54.866523	11
2	4	Komentarz testowy #2.	2026-06-17 15:42:57.309526	11
3	11	Komentarz testowy #3.	2026-06-18 18:23:23.1402	112
4	8	Komentarz testowy #4.	2026-06-18 18:24:35.297928	112
5	12	Komentarz testowy #5.	2026-06-18 18:34:59.042724	11
\.


ALTER TABLE public.service_desk_request_comments ENABLE TRIGGER ALL;

--
-- Data for Name: service_desk_requests; Type: TABLE DATA; Schema: public; Owner: -
--

ALTER TABLE public.service_desk_requests DISABLE TRIGGER ALL;

COPY public.service_desk_requests (id, title, description, type, status, created_by, created_by_id, assigned_to_id, created_at, updated_at, priority, location, location_id) FROM stdin;
2	Zgłoszenie #2 (other)	Opis zgłoszenia testowego #2.	other	resolved	Joanna Nowak	23	106	2026-06-16 14:32:26.606253	2026-06-16 14:47:17.692247	low	\N	6
1	Zgłoszenie #1 (other)	Opis zgłoszenia testowego #1.	other	resolved	Joanna Nowak	23	\N	2026-06-16 14:17:26.864967	2026-06-17 12:53:29.660444	high	\N	5
3	Zgłoszenie #3 (technical_problem)	Opis zgłoszenia testowego #3.	technical_problem	resolved	Joanna Nowak	112	11	2026-06-17 13:47:38.468765	2026-06-17 15:42:34.553001	low	\N	1
7	Zgłoszenie #7 (hardware_issue)	Opis zgłoszenia testowego #7.	hardware_issue	resolved	Joanna Nowak	9	13	2026-06-18 08:20:50.826865	2026-06-18 09:25:12.335204	high	\N	4
16	Zgłoszenie #16 (other)	Opis zgłoszenia testowego #16.	other	resolved	Joanna Nowak	11	1	2026-06-20 12:51:20.469819	2026-06-21 10:26:49.675489	high	\N	26
4	Zgłoszenie #4 (other)	Opis zgłoszenia testowego #4.	other	resolved	Joanna Nowak	11	2	2026-06-17 15:30:37.197333	2026-06-18 13:28:25.903842	high	\N	22
10	Zgłoszenie #10 (technical_problem)	Opis zgłoszenia testowego #10.	technical_problem	resolved	Joanna Nowak	9	\N	2026-06-18 16:36:11.315468	2026-06-18 18:11:14.977304	high	\N	21
5	Zgłoszenie #5 (other)	Opis zgłoszenia testowego #5.	other	resolved	Joanna Nowak	9	\N	2026-06-17 17:36:32.972925	2026-06-18 18:12:11.27392	high	\N	2
11	Zgłoszenie #11 (technical_problem)	Opis zgłoszenia testowego #11.	technical_problem	resolved	Joanna Nowak	11	\N	2026-06-18 18:15:29.362304	2026-06-19 09:04:01.323541	high	\N	47
17	Zgłoszenie #17 (other)	Opis zgłoszenia testowego #17.	other	resolved	Joanna Nowak	9	\N	2026-06-21 11:20:59.062743	2026-06-24 18:43:55.36501	high	\N	49
8	Zgłoszenie #8 (replacement)	Opis zgłoszenia testowego #8.	replacement	resolved	Joanna Nowak	9	\N	2026-06-18 14:11:52.949067	2026-06-18 18:24:39.386883	medium	\N	26
12	Zgłoszenie #12 (replacement)	Opis zgłoszenia testowego #12.	replacement	resolved	Joanna Nowak	11	\N	2026-06-18 18:30:26.774629	2026-06-19 09:37:39.438481	high	\N	1
13	Zgłoszenie #13 (other)	Opis zgłoszenia testowego #13.	other	resolved	Joanna Nowak	11	\N	2026-06-18 19:19:50.48572	2026-06-19 18:48:17.636514	high	\N	25
14	Zgłoszenie #14 (hardware_issue)	Opis zgłoszenia testowego #14.	hardware_issue	resolved	Joanna Nowak	9	\N	2026-06-19 18:49:47.660193	2026-06-19 18:50:15.246514	high	\N	2
6	Zgłoszenie #6 (other)	Opis zgłoszenia testowego #6.	other	resolved	Joanna Nowak	9	\N	2026-06-17 20:48:13.194555	2026-06-19 18:50:27.902888	high	\N	32
9	Zgłoszenie #9 (technical_problem)	Opis zgłoszenia testowego #9.	technical_problem	resolved	Joanna Nowak	11	\N	2026-06-18 16:08:48.056832	2026-06-19 18:55:16.94884	high	\N	3
15	Zgłoszenie #15 (other)	Opis zgłoszenia testowego #15.	other	resolved	Joanna Nowak	72	8	2026-06-19 23:43:05.427859	2026-06-20 06:36:52.577989	high	\N	22
\.


ALTER TABLE public.service_desk_requests ENABLE TRIGGER ALL;

--
-- Data for Name: transfer_users; Type: TABLE DATA; Schema: public; Owner: -
--

ALTER TABLE public.transfer_users DISABLE TRIGGER ALL;

COPY public.transfer_users (id, transfer_id, user_id) FROM stdin;
1	1	23
2	2	66
3	3	9
4	4	11
5	5	69
6	6	11
7	10	11
8	28	68
9	28	11
10	32	71
11	33	106
12	34	23
13	50	8
14	48	8
15	47	8
16	51	71
17	36	106
18	38	106
19	52	10
20	53	9
21	54	13
22	55	71
23	56	13
24	57	11
25	57	72
26	58	11
27	70	66
28	71	66
29	73	69
31	88	71
32	89	13
33	89	72
34	106	2
35	108	11
36	116	2
37	119	68
38	120	13
39	121	11
40	127	3
41	154	108
42	155	106
43	156	66
\.


ALTER TABLE public.transfer_users ENABLE TRIGGER ALL;

--
-- Name: audit_logs_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.audit_logs_id_seq', 2260, true);


--
-- Name: equipment_request_category_mapping_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.equipment_request_category_mapping_id_seq', 1, false);


--
-- Name: equipment_request_items_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.equipment_request_items_id_seq', 254, true);


--
-- Name: equipment_request_location_mapping_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.equipment_request_location_mapping_id_seq', 23, true);


--
-- Name: equipment_request_price_list_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.equipment_request_price_list_id_seq', 101879, true);


--
-- Name: equipment_request_quests_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.equipment_request_quests_id_seq', 95, true);


--
-- Name: equipment_request_sync_log_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.equipment_request_sync_log_id_seq', 7410, true);


--
-- Name: item_category_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.item_category_id_seq', 47, true);


--
-- Name: items_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.items_id_seq', 305, true);


--
-- Name: locations_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.locations_id_seq', 57, true);


--
-- Name: non_serialized_items_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.non_serialized_items_id_seq', 183, true);


--
-- Name: non_serialized_transfers_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.non_serialized_transfers_id_seq', 181, true);


--
-- Name: origins_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.origins_id_seq', 10, true);


--
-- Name: pyr_code_reservations_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.pyr_code_reservations_id_seq', 57, true);


--
-- Name: quest_transfers_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.quest_transfers_id_seq', 82, true);


--
-- Name: release_assets_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.release_assets_id_seq', 201, true);


--
-- Name: release_stocks_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.release_stocks_id_seq', 76, true);


--
-- Name: releases_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.releases_id_seq', 6, true);


--
-- Name: schedule_assignments_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.schedule_assignments_id_seq', 32040, true);


--
-- Name: schedule_day_windows_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.schedule_day_windows_id_seq', 3, true);


--
-- Name: schedule_slots_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.schedule_slots_id_seq', 339, true);


--
-- Name: schedule_volunteers_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.schedule_volunteers_id_seq', 207, true);


--
-- Name: schedules_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.schedules_id_seq', 10, true);


--
-- Name: serialized_transfers_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.serialized_transfers_id_seq', 547, true);


--
-- Name: service_desk_request_comments_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.service_desk_request_comments_id_seq', 5, true);


--
-- Name: service_desk_requests_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.service_desk_requests_id_seq', 17, true);


--
-- Name: transfer_users_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.transfer_users_id_seq', 43, true);


--
-- Name: transfers_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.transfers_id_seq', 257, true);


--
-- Name: users_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.users_id_seq', 114, true);


--
-- PostgreSQL database dump complete
--

\unrestrict fhfvY2tZyL43oPSniNxyy5rL0KXQ0gwYZr2sCcFRlW4uynHHNzPREFgPp4gJfzq

