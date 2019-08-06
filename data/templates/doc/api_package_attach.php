<div id="documentation">

  <ol class="breadcrumb">
    <li>Documentations</li>
    <li><a href="<?=url('/doc/api')?>">API</a></li>
    <li class="active">Attach File</li>
  </ol>

  <div class="page-header">
    <h2><i class="fa fa-upload"></i> Attach File API</h2>
  </div>

  <div>
    <h3 class="subheader">Request</h3>
    <div class="container">
      <dl class="dl-horizontal">
        <dt>HTTP Method:</dt>
        <dd>POST</dd>
        <dt>URL:</dt>
        <dd><a href="#"><?=url('/api/package_attach')?></a></dd>
        <dt>Response format:</dt>
        <dd>json</dd>
        <dt>Fiels:</dt>
        <dd>
          <ul>
            <li><code>api_key</code> - Required (See the application preferences)</li>
            <li><code>id</code> - Required, the package id</li>
            <li><code>file</code> - Required, the file to attach</li>
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
  "id": "334",
  "original_file_name": "testfile.txt",
  "file_size": 3170,
  "created": "2019-08-05 20:34:53"
}</code></pre>
        </dd>
      </dl>
    </div>
  </div>

  <div>
    <h3 class="subheader">Sample Curl</h3>
    <div class="container">
      <pre><code>curl <?=url('/api/package_attach')?> \
  -F api_key='{application_api_key}' \
  -F id=1 \
  -F file=@testfile.txt</code></pre>
    </div>
  </div>

</div>