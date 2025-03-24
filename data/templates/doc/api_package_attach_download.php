<div id="documentation">

  <ol class="breadcrumb">
    <li>Documents</li>
    <li><a href="<?=url('/doc/api')?>">API</a></li>
    <li class="active">Download Attach File</li>
  </ol>

  <div class="page-header">
    <h2><i class="fa fa-upload"></i>Download Attach File API</h2>
  </div>

  <div>
    <h3 class="subheader">Request</h3>
    <div class="container">
      <dl class="dl-horizontal">
        <dt>HTTP Method:</dt>
        <dd>POST</dd>
        <dt>URL:</dt>
        <dd><a href="#"><?=url('/api/package_attach_download')?></a></dd>
        <dt>Response format:</dt>
        <dd>json</dd>
        <dt>Fields:</dt>
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
        <dd>302 Found</dd>
      </dl>
    </div>
  </div>

  <div>
    <h3 class="subheader">Sample Curl</h3>
    <div class="container">
      <pre><code>curl "<?=url('/api/package_attach_download?api_key={application_api_key}&id=1&attached_file_id=334')?>"</code></pre>
    </div>
  </div>

 <div>
    <h3 class="subheader">Caveats</h3>
    <div class="container">
      <p>The type of numeric fields (id, file_size, etc.) are indeterminate and can be either string or number.</p>
    </div>
 </div>

</div>
