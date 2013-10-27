
<div id="signin-dialog" class="col-sm-6 col-sm-offset-3">
  <div class="panel panel-default">
    <div class="panel-heading">
      <h2 class="panel-title">Login</h2>
    </div>

    <div class="panel-body">
      <?php if($enable_password):?>
      <form class="form-horizontal" method="post" action="#">

        <div class="form-group">
          <label class="control-label col-sm-3" for="name">email</label>
          <div class="col-sm-9">
            <input class="form-control" type="text" id="name" name="name">
          </div>
        </div>
        <div class="form-group">
          <label class="control-label col-sm-3" for="password">password</label>
          <div class="col-sm-9">
            <input class="form-control" type="password" id="password" name="password">
          </div>
        </div>
        <div class="col-sm-9 col-sm-offset-3">
          <input type="submit" class="btn btn-primary" value="login">
          <a class="btn btn-link" href="">forget password</a>
        </div>
      </form>
      <?php endif ?>

      <?php if($enable_google_auth): ?>
      <div class="google-login col-sm-10 col-sm-offset-1">
        <a class="btn btn-primary col-xs-12">Login with google account</a>
      </div>
      <?php endif ?>
    </div>
  </div>
</div>
