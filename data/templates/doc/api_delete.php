<div id="documentation">

  <ol class="breadcrumb">
    <li>Documentations</li>
    <li><a href="<?=url('/doc/api')?>">API</a></li>
    <li class="active">Delete</li>
  </ol>

  <div class="page-header">
    <h2><i class="fa fa-trash-o"></i> Delete API</h2>
  </div>

  <div>
    <h3 class="subheader">Request</h3>
    <div class="container">
      <dl class="dl-horizontal">
        <dt>HTTP Method:</dt>
        <dd>GET</dd>
        <dt>URL:</dt>
        <dd><a href="#"><?=url('/api/delete')?></a></dd>
        <dt>Response format:</dt>
        <dd>json</dd>
        <dt>Fiels:</dt>
        <dd>
          <ul>
            <li><code>api_key</code> - Required (See the application preferences)</li>
            <li><code>id</code> - Required, package id to delete</li>
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
  "package_url": "<?=url("/package?id=3")?>",
  "application_url": "<?=url("/app?id=1")?>",
  "id": "3",
  "platform": "Android",
  "title": "test upload",
  "description": "upload package via upload api",
  "identifier": "org.klab.emlauncher.sample",
  "original_file_name": "emlauncher.apk",
  "file_size": "5776313",
  "created": "2013-11-29 12:26:19",
  "tags": [
    "test",
    "upload-api",
    "android"
  ],
  "install_count": 0
}</code></pre>
        </dd>
      </dl>
    </div>
  </div>

 <div>
    <h3 class="subheader">Sample Curl</h3>
    <div class="container">
      <pre><code>curl <?=url('/api/delete?api_key={application_api_key}&id={package_id}')?></code></pre>
    </div>
 </div>

</div>