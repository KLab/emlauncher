<div id="documentation">

  <ol class="breadcrumb">
    <li>Documentations</li>
    <li><a href="<?=url('/doc/api')?>">API</a></li>
    <li class="active">Upload</li>
  </ol>

  <div class="page-header">
    <h2><i class="fa fa-upload"></i> Upload API</h2>
  </div>

  <div>
    <h3 class="subheader">Request</h3>
    <div class="container">
      <dl class="dl-horizontal">
        <dt>HTTP Method:</dt>
        <dd>POST</dd>
        <dt>URL:</dt>
        <dd><a href="#"><?=url('/api/upload')?></a></dd>
        <dt>Response format:</dt>
        <dd>json</dd>
        <dt>Fiels:</dt>
        <dd>
          <ul>
            <li><code>api_key</code> - Required (See the application preferences)</li>
            <li><code>file</code> - Required, package file data</li>
            <li><code>title</code> - Required, title of the package</li>
            <li><code>description</code> - Optional, description of the package</li>
            <li><code>tags</code> - Optional, comma separated tag names</li>
            <li><code>protect</code> - Optional, protect from auto delete (default to False)</li>
            <li><code>notify</code> - Optional, notify application-installed users (defaults to False)</li>
            <li><code>dsym</code> - Optional, attach dSYM file</li>
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
  "package_url": "<?=url("/package?id=17")?>",
  "application_url": "<?=url("/app?id=2")?>",
  "id": "17",
  "platform": "Android",
  "title": "aab upload",
  "description": "upload aab package via upload api",
  "identifier": "com.example.android.basicaccessibility",
  "original_file_name": "Application.aab",
  "file_size": 1339565,
  "protect": true,
  "created": "2019-07-08 03:22:00",
  "tags": [
    "test",
    "upload-api",
    "aab"
  ],
  "install_count": 0,
  "attached_files": [
    {
      "id": "28",
      "original_file_name": "Application.apk",
      "file_size": "1452918",
      "created": "2019-07-08 03:22:00"
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
      <pre><code>curl <?=url('/api/upload')?> \
  -F api_key='{application_api_key}' \
  -F file=@Application.aab \
  -F title='aab upload' \
  -F description='upload aab package via upload api' \
  -F tags='test,upload-api,aab' \
  -F protect=True \
  -F notify=False</code></pre>
    </div>
  </div>

</div>