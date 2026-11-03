<div id="documentation">

  <ol class="breadcrumb">
    <li>Documents</li>
    <li class="active">API</li>
  </ol>

  <div class="page-header">
    <h2><i class="fa fa-puzzle-piece"></i> API</h2>
  </div>

  <div>
    <h3 class="subheader">Authentication</h3>
    <div class="container">
      <p>All APIs require the API key of the application (See the application preferences).</p>
      <p>Send it in the <code>Authorization</code> request header (recommended):</p>
      <pre><code>Authorization: Bearer {application_api_key}</code></pre>
      <p>Sending it as the <code>api_key</code> request field (query string or form data) is still accepted for backward compatibility, but deprecated. A query string is recorded in the access logs of web servers and proxies, and the API key may be exposed.</p>
      <p>If both are specified, the header takes precedence.</p>
      <p>The administrator can reject the <code>api_key</code> request field by setting <code>api_key_param_enabled</code> to <code>false</code> in the configuration. In that case, a request without the header results in a 400 error.</p>
    </div>
  </div>

  <div class="list-group">
    <a href="<?=url('/doc/api_upload')?>" class="list-group-item">
      <h3 class="list-group-item-heading">Upload API</h3>
      <p class="list-gorup-item-text">Upload a package file.</p>
    </a>
    <a href="<?=url('/doc/api_package_list')?>" class="list-group-item">
      <h3 class="list-group-item-heading">Package List API</h3>
      <p class="list-gorup-item-text">Get package list.</p>
    </a>
    <a href="<?=url('/doc/api_package_edit')?>" class="list-group-item">
      <h3 class="list-group-item-heading">Package Edit API</h3>
      <p class="list-gorup-item-text">Edit package information.</p>
    </a>
    <a href="<?=url('/doc/api_delete')?>" class="list-group-item">
      <h3 class="list-group-item-heading">Delete API</h3>
      <p class="list-gorup-item-text">Delete a package.</p>
    </a>
    <a href="<?=url('/doc/api_create_token')?>" class="list-group-item">
      <h3 class="list-group-item-heading">Create Token API</h3>
      <p class="list-gorup-item-text">Create Token and return install url.</p>
    </a>
    <a href="<?=url('/doc/api_package_attach')?>" class="list-group-item">
      <h3 class="list-group-item-heading">Attach File API</h3>
      <p class="list-gorup-item-text">Attach a file to the package.</p>
    </a>
    <a href="<?=url('/doc/api_package_detach')?>" class="list-group-item">
      <h3 class="list-group-item-heading">Detach File API</h3>
      <p class="list-gorup-item-text">Delete the attached file from the package.</p>
    </a>
    <a href="<?=url('/doc/api_package_attach_download')?>" class="list-group-item">
      <h3 class="list-group-item-heading">Attach File Download API</h3>
      <p class="list-gorup-item-text">Get a link to the attached file.</p>
    </a>
  </div>


</div>
