# Go baseline migration manifest

This document is generated from the reviewed Laravel migration source tree.
It defines the immutable baseline that `idelium-api-go` will use during
schema ownership handover. The baseline is intentionally not applied by
this ticket; Laravel remains the schema owner until the bridge, empty
install, upgrade, cutover, and rollback gates pass.

## Summary

- Baseline ID: `go-baseline-2026-08-25`
- Generated on: `2026-08-25`
- Source runtime: idelium-api Laravel migrations
- Target runtime: idelium-api-go
- Migration count: 69
- Aggregate SHA-256: `f54df2c9f302d89231b127deaef226e4f37cd0e1058c7db4f3ed218f3c886698`
- Review status: `review-required-before-apply`

## Handover policy

- Laravel remains the schema owner during coexistence.
- Go baseline application is disabled until the Wave 10 bridge and
  verification tickets are complete.
- Dual writes are not allowed.
- Rollback is a Git revert while this manifest is review-only; after
  schema handover, rollback must use the documented last dual-runtime
  release path.

## Redaction

Migration source hashes and file sizes are recorded; no tenant data, credentials, or payload values are included.

## Included Laravel migrations

| # | Migration | SHA-256 | Size |
| ---: | --- | --- | ---: |
| 1 | `2014_10_12_000000_create_users_table.php` | `94797e63add6c1604bfd0a2a4f6ec15f0f4260008c0f4c3675c60fe96a79adb1` | 835 |
| 2 | `2014_10_12_100000_create_password_resets_table.php` | `4523ad86b515b1d37c4e182e12973247b793e50511f1065d8ef9d57fac71c45d` | 683 |
| 3 | `2019_08_19_000000_create_failed_jobs_table.php` | `b7ff81d0a2324199c07ef4141748a839a73b04b01bfe6d32bda6635987136fae` | 820 |
| 4 | `2019_12_14_000001_create_personal_access_tokens_table.php` | `bef9afbe8608f332c82a4b8a450918b94cf69aa1dd544acbee72d3e8d26bd3ea` | 933 |
| 5 | `2021_08_23_114345_create_projects_table.php` | `d1288c35dd14289e9010ab63ea8a322ed2889c9cc7833f5482cd1610d61dacb3` | 662 |
| 6 | `2021_08_23_151631_create_environments_table.php` | `a923114a88da6850ec5dbae325a4514d09a541679737895ac9cb8f9f0b2fc901` | 752 |
| 7 | `2021_08_24_071355_create_plugins_table.php` | `128614b6f9146e640f723f7ace1f3b8e059751ef980ba666779b9bd4f6ebeea2` | 735 |
| 8 | `2021_08_24_091348_create_steps_table.php` | `05d8b0f774a6bd5b8e765a8565781a16b257e2ebbac9469b2e3d80ed3becc070` | 769 |
| 9 | `2021_08_24_114022_create_tests_table.php` | `5bb8bcf05b72183ed14332674226eaf55040e9fbba5f54d97b729ccf6431f27e` | 731 |
| 10 | `2021_08_24_124842_create_test_cycles_table.php` | `51e97b4731c0b0786c27bec2aedbe618a0f4aa1f8f341222687ff3400782aeca` | 748 |
| 11 | `2021_08_26_081929_create_performed_test_cycles_table.php` | `fdaca6144f4a64909f89c89e5770013d0c373a0a386de76280065f0663972827` | 741 |
| 12 | `2021_08_26_082013_create_performed_tests_table.php` | `5e2569d3b1d5c7806860e140f16b0b02ca76cdad702d3d30a6841c285e633c48` | 765 |
| 13 | `2021_08_26_082805_create_performed_steps_table.php` | `912b55feac60287a2ff5421cff4e2767cc353c8285813901f82990a071927b4b` | 849 |
| 14 | `2021_08_31_120646_create_api_keys_table.php` | `11617f91f6366d5a99e0ebf5c2dd60d31b02fc93744430ba7ddecbdac1f851be` | 745 |
| 15 | `2021_08_31_122956_create_roles_table.php` | `0e46545c05842d788414abe65386836b068a475f3f75b9d1b3cdc765aec825ba` | 610 |
| 16 | `2021_09_01_094727_create_costumers_table.php` | `d41f1f9934a80ff32c143ee18adffa2cd67ceb25eab3d9937922ffc32cb644bf` | 754 |
| 17 | `2021_09_01_100641_add_id_costumer_to_projects.php` | `8ce1482bd61117cf411819f33bb9d754f01657bf3022467b40d356f21d3508f1` | 650 |
| 18 | `2021_09_01_100848_add_id_costumer_to_users.php` | `3d67bac87c8a3c099ee452deaf361f4002ac16498363cef4a947137826d0d4fe` | 640 |
| 19 | `2021_09_01_121412_add_apikey_to_costumers.php` | `32500a54244656c5196125b335a45394360c44a989cb26486487ddfe7cbb8456` | 663 |
| 20 | `2021_09_01_123652_create_sessions_table.php` | `d9de0228f1ed2c2eca5454ea01c12978c3bc88718a38a05e5ddd0dfdf8c92294` | 833 |
| 21 | `2021_09_01_125201_add_id_costumer_to_steps.php` | `9c4eea86f06d2381a40f7c7cd1485fe5d675f788b783d8e631901fc4bafea039` | 640 |
| 22 | `2021_09_01_125247_add_id_costumer_to_tests.php` | `ac8069cc5b7ab7fcdad92fa3868ad1821908b773594e0c740a60db74cdce0bb0` | 640 |
| 23 | `2021_09_01_125306_add_id_costumer_to_testcycles.php` | `c6963957d4ab025882bc7f387eb6fac9ff15d954a0531c0bdfacbf578a164a97` | 657 |
| 24 | `2021_09_01_125347_add_id_costumer_to_plugins.php` | `b2b718c05b67aeeba453f7dffd6f5bae1d4094ab49db097980defcb7a4024ac6` | 646 |
| 25 | `2021_09_01_125450_add_id_costumer_to_performedsteps.php` | `c833fea1c9ecdc197682d3eda9b9a739429d46fb7b7b0cf7664ceae658ff72d3` | 669 |
| 26 | `2021_09_01_125533_add_id_costumer_to_performedtests.php` | `e81e4d25a0c0078da94cb5f9173056c598fe86cd0edc287fe51de0215a7f6e44` | 669 |
| 27 | `2021_09_01_125553_add_id_costumer_to_performedtestcycles.php` | `9dc378827b91c86335cf88aca8ba0b4435ff6ea2398868f7149208290321db6f` | 686 |
| 28 | `2021_09_01_132929_add_id_costumer_to_environments.php` | `0d6466699d6bb70df29a10914957dfe12fb304838a47d4c67746febfd74d9766` | 661 |
| 29 | `2021_10_14_000000_add_provider_id_name_to_users.php` | `cb310c380db5733c75ec6be6085b8e1282c097d1d8559a9358120b798b166c12` | 763 |
| 30 | `2022_01_03_092454_create_locations_table.php` | `fd05fa29ef5a96acd5dc3b9702a591bdec70b7413474e731b8d0256061d13725` | 622 |
| 31 | `2022_01_03_100904_create_browsers_table.php` | `9cf12d70ea4f0c2377fe70fde30ef1b750a24da9d30c85368a28006f64bc5321` | 656 |
| 32 | `2022_01_03_100919_create_os_table.php` | `24214e93c1bd8b387b4134204a28960ec5a22cf3197c990d45e39416ff8956f2` | 638 |
| 33 | `2022_01_03_100936_create_types_table.php` | `45620f1e5fd357157f20813914613ff23434f57ab85ed4ced2d25acf616c404f` | 610 |
| 34 | `2022_01_03_100946_create_descriptions_table.php` | `c94a69019f0ed5c14a81b15744efef29ec8e6e37e29154467cc0208b5934adc3` | 638 |
| 35 | `2022_01_03_100958_create_hostnames_table.php` | `774f7f9ac64510ede12b94953f294f77c38a56fc0e28df5bd0a41a6199f5d4ef` | 669 |
| 36 | `2022_01_03_101030_create_version_browsers_table.php` | `69d66d234b1683e0d785f6a0d4ead391931406ff953f7cd59ed6d443bf565196` | 687 |
| 37 | `2022_01_03_101408_create_brand_devices_table.php` | `aba0d292c8c503c80f85f6eb30fecd16d01c1a1c36c8c19035dfefe0425fb895` | 634 |
| 38 | `2022_01_03_101819_create_version_os_table.php` | `83868fc5c9e69aaed97474e6b96f7e14959bc1dadb238db1daa9fb9b8789100c` | 664 |
| 39 | `2022_01_05_112123_add_id_o_s_browsers.php` | `515ced4af9fd5b56c72dc6aa2ec620fc50f6f9237235c480c546839d7cf9db1e` | 663 |
| 40 | `2022_01_08_090421_create_model_devices_table.php` | `90ae40df26c88eea71515881176cfbae1692c2eaf4119f6641ad4c3b8568eb21` | 743 |
| 41 | `2022_01_08_112254_create_platforms_table.php` | `7dc2553f09ee92a1da7934164c756bd9b7bb05967444fe9fd1fb86dd68415156` | 1084 |
| 42 | `2022_01_11_112155_create_statuses_table.php` | `65db900492f9a70c7392a73ca3e0f04be61e7ec1a14ce5d85099012738161126` | 661 |
| 43 | `2022_08_05_132929_add_data_type_to_performed_step.php` | `1e7e54a67337952da063e72b8f2098e5f86ed4f25e4f6716adbff1a220b2378e` | 727 |
| 44 | `2026_07_15_120000_add_core_mutation_foreign_keys.php` | `162f9418e07334014e901f17765356bc3b96658971800377963a68cb5111c36b` | 2472 |
| 45 | `2026_07_27_120000_create_parallel_run_schedules_table.php` | `8b31bcdc7338038e264bd3ab51c94b29130a65a066a4f7c45f67fcdcd6015605` | 2333 |
| 46 | `2026_07_28_120000_create_audit_events_table.php` | `465a6c030c68e802664c16046faddcc297e842d8d9cd13ef844770d715b02513` | 1771 |
| 47 | `2026_07_28_130000_create_artifact_descriptors_table.php` | `e12f03bbcc9d77fd760d79ca44a570f465cf5882cacda30441ff6e9c59b10419` | 2154 |
| 48 | `2026_07_28_140000_create_service_accounts_table.php` | `4fd1158c51d335639f2c939e3c4a9daa1fac3029d71d60afcdb25f996078cce4` | 1208 |
| 49 | `2026_07_28_150000_add_result_exploration_indexes.php` | `d9770b44174abaaa246f5723fae88e991e019fe64dbe443983d58b425845c100` | 1060 |
| 50 | `2026_07_28_150000_create_run_tokens_table.php` | `dc6cedeab6749e1209cf4bfcd370e7fb4bcdfb1fc97a59da284f5125c03f3018` | 1396 |
| 51 | `2026_07_28_151000_create_result_exports_table.php` | `d6035b0a4cb1bea7d84e086492c4f1b7c7c8a876cb13582be4dfbd11dfebc859` | 1303 |
| 52 | `2026_07_28_160000_create_asset_versions_table.php` | `433b14236900d645209d5acdee1873d27a59b181733c59e09df08b5584484e4a` | 1355 |
| 53 | `2026_07_28_170000_create_asset_version_review_events_table.php` | `2dc3f3d6e531b1f92fdfd9ef4bde5d7c9aaf393293c3057f51a00d79704d5d7c` | 1398 |
| 54 | `2026_07_28_180000_create_agent_registrations_table.php` | `3af03de7128fd4164ed335d14a84b60813743d6bdf1aa0c7ccbd26d514a78ba4` | 1214 |
| 55 | `2026_07_28_190000_create_oidc_workload_tokens_table.php` | `c5a300e366f0093b23becac814fd0b6d1679a517e579794c4f9909161e2f44fc` | 1926 |
| 56 | `2026_07_28_200000_add_identity_proof_to_agent_registrations_table.php` | `669da902407044e40d7f164c93849e52af89c0c46b11c60e65b51f34eab8eaee` | 565 |
| 57 | `2026_07_28_210000_create_integration_endpoints_table.php` | `2a656525df3d53dddacaaf535e26dae8c0bc652a5c4ffc1482f3346c360e458c` | 1308 |
| 58 | `2026_07_28_211000_create_integration_deliveries_table.php` | `b7ce1d48f8f47b7c3246a110c0c10b2be5dc1ada72349023a41cbfa04e54154a` | 1899 |
| 59 | `2026_07_28_220000_create_identity_providers_table.php` | `7e3d599e48291fe87f9fae59601ce4543eaa652521c7cdae339d47e42cab87ee` | 1221 |
| 60 | `2026_07_28_221000_create_scim_identities_table.php` | `8b15828e11765825719fa8aee6299f4b408d3491e4929ff8878eabda5bfd8c91` | 1392 |
| 61 | `2026_07_28_222000_add_identity_lifecycle_columns_to_users_table.php` | `6afa1918821f959cd64203a9c0eabed6bce759e78e5c1e309aab777ebe68b45c` | 1321 |
| 62 | `2026_07_28_223000_add_mfa_state_to_users_table.php` | `7d007f4693ac74358461f3ec903a70e4131b36661a59367f41a16c03edc18bb7` | 844 |
| 63 | `2026_07_29_090000_create_grid_bulk_operation_tables.php` | `da0e77d99ecff90fdc6a107227703549f5484066fa2c9e136f905f3954b073ce` | 2713 |
| 64 | `2026_07_30_123600_add_postman_data_to_performed_tests.php` | `cd9c188061e6cd6b8886a0ab93d4730de7dc88d2c255f34f25299610ce7f1ef6` | 732 |
| 65 | `2026_08_07_090000_add_legacy_api_key_usage_to_costumers.php` | `7e8b1518380ecb943b7b188ae6766dfccec627154717ca291f05bb9060bf8eb4` | 871 |
| 66 | `2026_08_07_100000_ensure_legacy_api_key_expiration_column.php` | `3c0fbebbe12776f9060024b6bfad47cd1f812893e1d8249a257b9e72253b4014` | 764 |
| 67 | `2026_08_27_120000_add_idempotency_keys_to_performed_results.php` | `43e7a34c57c41db587efafc108919f902e98c2f6fd7e620c410c990436f57e1c` | 1020 |
| 68 | `2026_08_27_130000_create_go_browser_sessions_table.php` | `2d6fb7a4fd32b19dab388b4f551ab1fce93d6e57585034273866a193e774afb7` | 991 |
| 69 | `2026_08_27_140000_add_active_tenant_to_go_browser_sessions_table.php` | `a1e5d2d1fa47107fba38794a59d968867228e547924f78843a784d89f74d15ca` | 1236 |

## Regeneration

```sh
python3 scripts/build_go_baseline_migration.py \
  --source-dir ../idelium-api/database/migrations \
  --output-json docs/contracts/go-baseline-migration.json \
  --output-markdown docs/contracts/go-baseline-migration.md
```
