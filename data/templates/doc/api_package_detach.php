<div id="documentation">

  <ol class="breadcrumb">
    <li>Documentations</li>
    <li><a href="<?=url('/doc/api')?>">API</a></li>
    <li class="active">Detach File</li>
  </ol>

  <div class="page-header">
    <h2><i class="fa fa-trash-o"></i> Detach File API</h2>
  </div>

  <div>
    <h3 class="subheader">Request</h3>
    <div class="container">
      <dl class="dl-horizontal">
        <dt>HTTP Method:</dt>
        <dd>GET</dd>
        <dt>URL:</dt>
        <dd><a href="#"><?=url('/api/package_detach')?></a></dd>
        <dt>Response format:</dt>
        <dd>json</dd>
        <dt>Fiels:</dt>
        <dd>
          <ul>
            <li><code>api_key</code> - Required (See the application preferences)</li>
            <li><code>id</code> - Required, the package id</li>
            <li><code>attached_file_id</code> - Required, the attached file id to delete</li>
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
      <pre><code>curl <?=url('/api/package_detach?api_key={application_api_key}&id=1&attached_file_id=334')?></code></pre>
    </div>
 </div>

</div>