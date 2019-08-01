<div id="documentation">

  <ol class="breadcrumb">
    <li>Documentations</li>
    <li><a href="<?=url('/doc/api')?>">API</a></li>
    <li class="active">Package List</li>
  </ol>

  <div class="page-header">
    <h2><i class="fa fa-list-ul"></i> Package List API</h2>
  </div>

  <div>
    <h3 class="subheader">Request</h3>
    <div class="container">
      <dl class="dl-horizontal">
        <dt>HTTP Method:</dt>
        <dd>GET</dd>
        <dt>URL:</dt>
        <dd><a href="#"><?=url('/api/package_list')?></a></dd>
        <dt>Response format:</dt>
        <dd>json</dd>
        <dt>Fields:</dt>
        <dd>
          <ul>
            <li><code>api_key</code> - Required (See the application preferences)</li>
            <li><code>limit</code> - Optional, The limit length of package list. (default = 20, max = 100)</li>
            <li><code>offset</code> - Optional, The offset of package list. (default = 0)</li>
            <li><code>platform</code> - Optional, The platform type string, e.g. "Android" and "iOS".</li>
            <li><code>tags</code> - Optional, The comma separated tag names to filter package list. These are treated as AND condition.</li>
          </ul>
        </dd>
      </dl>
    </div>
  </div>

  <div>
    <h3 class="subheader">Response</h3>
    <div class="container">
      <dl class="dl-horizontal">
        <dt>Status:</dt>
        <dd>200 OK</dd>
        <dt>Sample response:</dt>
        <dd>
          <pre><code>[
  {
    "package_url": "<?=url("/package?id=176825")?>"
    "application_url": "<?=url("/app?id=1")?>",
    "id": "176825",
    "platform": "Android",
    "title": "test aab",
    "description": "",
    "identifier": "com.example.android.basicaccessibility",
    "original_file_name": "Application.aab",
    "file_size": "1339565",
    "protect": true,
    "created": "2019-07-20 19:05:50",
    "tags": [
      "Android",
      "aab"
    ],
    "install_count": 0,
    "attached_files": [
      {
        "id": "3",
        "original_file_name": "Application.apk",
        "file_size": "1452918",
        "created": "2019-07-20 19:05:50"
      }
    ]
  },
  {
    "package_url": "<?=url("/package?id=2")?>",
    "application_url": "<?=url("/app?id=1")?>",
    "id": "2",
    "platform": "iOS",
    "title": "iPhone用",
    "description": "あいほんようipaふぁいる                    ",
    "identifier": "com.klab.playground-sandboxes.test6",
    "original_file_name": null,
    "file_size": null,
    "protect": false,
    "created": "2013-11-21 17:05:55",
    "tags": [
      "installable",
      "iOS"
    ],
    "install_count": 13,
    "attached_files": []
  },
  {
    "package_url": "<?=url("/package?id=1")?>",
    "application_url": "<?=url("/app?id=1")?>",
    "id": "1",
    "platform": "Android",
    "title": "Android用",
    "description": "あんどろいどようapkふぁいる\r\nまだ作りかけです",
    "identifier": "",
    "original_file_name": null,
    "file_size": null,
    "protect": false,
    "created": "2013-11-21 17:05:06",
    "tags": [
      "installable",
      "Android"
    ],
    "install_count": 2,
    "attached_files": []
  }
]</code></pre>

        </dd>
      </dl>
    </div>
  </div>

 <div>
    <h3 class="subheader">Sample Curl</h3>
    <div class="container">
      <pre><code>curl <?=url('/api/package_list?api_key={application_api_key}')?></code></pre>
    </div>
 </div>

</div>
