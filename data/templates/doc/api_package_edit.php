<div id="documentation">

  <ol class="breadcrumb">
    <li>Documentations</li>
    <li><a href="<?=url('/doc/api')?>">API</a></li>
    <li class="active">Package Edit</li>
  </ol>

  <div class="page-header">
    <h2><i class="fa fa-edit"></i> Package Edit API</h2>
  </div>

  <div>
    <h3 class="subheader">Request</h3>
    <div class="container">
      <dl class="dl-horizontal">
        <dt>HTTP Method:</dt>
        <dd>POST</dd>
        <dt>URL:</dt>
        <dd><a href="#"><?=url('/api/package_edit')?></a></dd>
        <dt>Response format:</dt>
        <dd>json</dd>
        <dt>Fields:</dt>
        <dd>
          <ul>
            <li><code>api_key</code> - Required (See the application preferences)</li>
            <li><code>id</code> - Required, Package id to edit</li>
            <li><code>title</code> - Optional, New title of the package.</li>
            <li><code>description</code> - Optional, New description of the package.</li>
            <li><code>protect</code> - Optional, New protect value of the package.</li>
            <li><code>tags</code> - Optional, New all tag names of the package (comma separated).</li>
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
          <pre><code>{
  "package_url": "<?=url("/package?id=176825")?>",
  "application_url": "<?=url("/app?id=1")?>",
  "id": "176825",
  "platform": "Android",
  "title": "aab test",
  "description": "",
  "identifier": "com.example.android.basicaccessibility",
  "original_file_name": "Application.aab",
  "file_size": "1339565",
  "protect": false,
  "created": "2019-07-20 19:05:50",
  "tags": [
    "Android",
    "aab",
    "test"
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
}</code></pre>
        </dd>
      </dl>
    </div>
  </div>

 <div>
    <h3 class="subheader">Sample Curl</h3>
    <div class="container">
      <pre><code>curl <?=url('/api/package_edit')?> \
  -F api_key='{application_api_key}' \
  -F id=176825 \
  -F tags=Android,aab,test \
  -F protect=False \
  -F title="aab test"</code></pre>
    </div>
 </div>

</div>
