<div id="documentation">

  <div class="page-header">
    <h2><i class="fa fa-upload"></i> Upload API</h2>
  </div>

  <div>
    <h3>Request</h3>
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
            <li><code>api_key</code> - Required (See application preference)</li>
            <li><code>file</code> - Required, package file data</li>
            <li><code>title</code> - Required, title of the package</li>
            <li><code>description</code> - Optional, description of the package</li>
            <li><code>tags</code> - Optional, comma separated tag names</li>
            <li><code>notify</code> - Optional, notify application-installed users (defaults to False)</li>
          </ul>
        </dd>
      </dl>
    </div>
  </div>

  <div>
    <h3>Response</h3>
    <div class="container">
      <dl class="dl-horizontal">
        <dt>Status:</dt>
        <dd>200 OK</dd>
        <dt>Sample response:</dt>
        <dd>
          <pre><code>{
aaaaaaaa,
bbbbbb,
}</code></pre>
        </dd>
      </dl>
    </div>
  </div>


  <div>
    <h3>Sample Curl</h3>
    <div class="container">
      <pre><code>curl <?=url('/api/upload')?>
  -F api_key='application_api_key'
  -F file=@emlauncher.pkg
  -F title='test upload'
  -F description='upload package via upload api'
  -F tags='test,upload-api,android'
  -F notify=True</code></pre>
    </div>
  </div>

</div>