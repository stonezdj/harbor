import unittest
import xmlrunner

if __name__ == "__main__":

    # find all required tests
    tests = unittest.TestSuite()
    tests.addTests(unittest.defaultTestLoader.discover('/drone/tests/apitests/python/', pattern='test_add_member_to_private_project.py'))
    tests.addTests(unittest.defaultTestLoader.discover('/drone/tests/apitests/python/', pattern='test_add_replication_rule.py'))
    tests.addTests(unittest.defaultTestLoader.discover('/drone/tests/apitests/python/', pattern='test_add_sys_label_to_tag'))
    tests.addTests(unittest.defaultTestLoader.discover('/drone/tests/apitests/python/', pattern='test_assign_sys_admin.py'))
    tests.addTests(unittest.defaultTestLoader.discover('/drone/tests/apitests/python/', pattern='test_banner_message.py'))
    tests.addTests(unittest.defaultTestLoader.discover('/drone/tests/apitests/python/', pattern='test_copy_artifact_outside_project.py'))
    tests.addTests(unittest.defaultTestLoader.discover('/drone/tests/apitests/python/', pattern='test_del_repo.py'))
    tests.addTests(unittest.defaultTestLoader.discover('/drone/tests/apitests/python/', pattern='test_edit_project_creation.py'))
    tests.addTests(unittest.defaultTestLoader.discover('/drone/tests/apitests/python/', pattern='test_heath_check.py'))
    tests.addTests(unittest.defaultTestLoader.discover('/drone/tests/apitests/python/', pattern='test_job_service_dashboard.py'))
    # tests.addTests(unittest.defaultTestLoader.discover('/drone/tests/apitests/python/', pattern='test_log_rotation.py')) unstable
    tests.addTests(unittest.defaultTestLoader.discover('/drone/tests/apitests/python/', pattern='test_manage_project_member.py'))
    tests.addTests(unittest.defaultTestLoader.discover('/drone/tests/apitests/python/', pattern='test_project_level_cve_allowlist'))
    # tests.addTests(unittest.defaultTestLoader.discover('/drone/tests/apitests/python/', pattern='test_project_permission.py')) #project_id type mismatch
    tests.addTests(unittest.defaultTestLoader.discover('/drone/tests/apitests/python/', pattern='test_project_quota.py'))
    # tests.addTests(unittest.defaultTestLoader.discover('/drone/tests/apitests/python/', pattern='test_proxy_cache.py')) x509 issue

    tests.addTests(unittest.defaultTestLoader.discover('/drone/tests/apitests/python/', pattern='test_quota_sorting.py'))
    tests.addTests(unittest.defaultTestLoader.discover('/drone/tests/apitests/python/', pattern='test_registry_api.py'))
    tests.addTests(unittest.defaultTestLoader.discover('/drone/tests/apitests/python/', pattern='test_retain_image_last_pull_time.py'))
    tests.addTests(unittest.defaultTestLoader.discover('/drone/tests/apitests/python/', pattern='test_retention.py'))
    tests.addTests(unittest.defaultTestLoader.discover('/drone/tests/apitests/python/', pattern='test_robot_account.py'))
    tests.addTests(unittest.defaultTestLoader.discover('/drone/tests/apitests/python/', pattern='test_sbom_generation_of_image_artifact.py'))
    tests.addTests(unittest.defaultTestLoader.discover('/drone/tests/apitests/python/', pattern='test_scan_data_export.py'))
    tests.addTests(unittest.defaultTestLoader.discover('/drone/tests/apitests/python/', pattern='test_scan_image_artifact_in_public_project.py'))
    tests.addTests(unittest.defaultTestLoader.discover('/drone/tests/apitests/python/', pattern='test_scan_image_artifact.py'))
    tests.addTests(unittest.defaultTestLoader.discover('/drone/tests/apitests/python/', pattern='test_security_hub.py'))
    tests.addTests(unittest.defaultTestLoader.discover('/drone/tests/apitests/python/', pattern='test_sys_cve_allowlists.py'))
    tests.addTests(unittest.defaultTestLoader.discover('/drone/tests/apitests/python/', pattern='test_sys_level_scan_all.py'))
    tests.addTests(unittest.defaultTestLoader.discover('/drone/tests/apitests/python/', pattern='test_user_view_logs.py'))
    tests.addTests(unittest.defaultTestLoader.discover('/drone/tests/apitests/python/', pattern='test_webhook_crud.py'))

    # generate xml report
    runner = xmlrunner.XMLTestRunner(output='test-reports')
    result = runner.run(tests)

    # exit with -1 if any test failed
    if not result.wasSuccessful():
        sys.exit(-1)