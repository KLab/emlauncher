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
        <dt>Fiels:</dt>
        <dd>
          <ul>
            <li><code>api_key</code> - Required (See the application preferences)</li>
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
    "package_url": "http://localhost/emlauncher/package?id=3",
    "application_url": "http://localhost/emlauncher/app?id=1",
    "id": "3",
    "platform": "Android",
    "title": "test upload",
    "description": "upload package via upload api",
    "ios_identifier": "",
    "original_file_name": "emlauncher.apk",
    "file_size": "5776313",
    "created": "2013-11-29 12:26:19",
    "tags": [
      "test",
      "upload-api",
      "android"
    ],
    "install_count": 1
  },
  {
    "package_url": "http://localhost/emlauncher/package?id=1",
    "application_url": "http://localhost/emlauncher/app?id=1",
    "id": "1",
    "platform": "iOS",
    "title": "ipa file test",
    "description": "test package for iPhone",
    "ios_identifier": "com.klab.playground-sandboxes.test6",
    "original_file_name": "emlauncher.ipa",
    "file_size": "4845763",
    "created": "2013-11-29 09:03:01",
    "tags": [
      "test",
      "ios"
    ],
    "install_count": 0
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