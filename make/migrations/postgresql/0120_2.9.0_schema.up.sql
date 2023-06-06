CREATE INDEX IF NOT EXISTS idx_task_extra_attrs_report_uuids ON task USING gin ((extra_attrs::jsonb->'report_uuids'));

/* Set the vendor_id of IMAGE_SCAN to the artifact id instead of scanner id, which facilitates execution sweep */
UPDATE execution SET vendor_id = (extra_attrs -> 'artifact' ->> 'id')::integer
WHERE jsonb_path_exists(extra_attrs::jsonb, '$.artifact.id')
AND vendor_id IN (SELECT id FROM scanner_registration)
AND vendor_type = 'IMAGE_SCAN';


/* extract score from vendor attribute */
update vulnerability_record
set cvss_score_v3 = cast ( vendor_attributes->'CVSS'->'nvd'->>'V3Score' as double precision)
where vendor_attributes->'CVSS'->'nvd'->>'V3Score' is not null;

alter table scan_report add column IF NOT EXISTS  critical_cnt int;
alter table scan_report add column IF NOT EXISTS  high_cnt int;
alter table scan_report add column IF NOT EXISTS  medium_cnt int;
alter table scan_report add column IF NOT EXISTS  low_cnt int;
alter table scan_report add column IF NOT EXISTS  none_cnt int;
alter table scan_report add column IF NOT EXISTS  unknown_cnt int;
alter table scan_report add column IF NOT EXISTS  fixable_cnt int;

/* extract summary information for previous scan_report */
DO
$$
    DECLARE
        report RECORD;
        v RECORD;
        critical_count int;
        high_count int;
        none_count int;
        medium_count int;
        low_count int;
        unknown_count int;
        fixable_count int;
    BEGIN
        FOR report IN SELECT * FROM scan_report
            LOOP
                critical_count := 0;
                high_count := 0;
                medium_count := 0;
                none_count := 0;
                low_count := 0;
                unknown_count := 0;
                FOR v IN SELECT vr.severity, vr.fixed_version from report_vulnerability_record rvr, vulnerability_record vr where rvr.report_uuid = report.uuid and rvr.vuln_record_id = vr.id
                    LOOP
                        if v.severity = 'Critical' then
                            critical_count = critical_count + 1;
                        end if;
                        if v.severity = 'High' then
                            high_count = high_count + 1;
                        end if;
                        if v.severity = 'Medium' then
                            medium_count = medium_count + 1;
                        end if;
                        if v.severity = 'Low' then
                            low_count = low_count + 1;
                        end if;
                        if v.severity = 'None' then
                            none_count = none_count + 1;
                        end if;
                        if v.severity = 'Unknown' then
                            unknown_count = unknown_count + 1;
                        end if;
                        if v.fixed_version is not null then
                            fixable_count = fixable_count + 1;
                        end if;
                    END LOOP;
                update scan_report set critical_cnt = critical_count, high_cnt = high_count, medium_cnt = medium_count, low_cnt = low_count, unknown_cnt = unknown_count where uuid = report.uuid;
            END LOOP;
    END
$$;
