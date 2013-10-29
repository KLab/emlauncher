
<div id="signin-dialog" class="col-sm-6 col-sm-offset-3">
  <div class="panel panel-primary">
    <div class="panel-heading">
      <h2 class="panel-title">Login</h2>
    </div>

    <div class="panel-body">
      <?php if($enable_password):?>
      <form class="form-horizontal" method="post" action="<?=url('/login/password')?>">

        <div class="form-group">
          <label class="control-label col-sm-3" for="email">email</label>
          <div class="col-sm-9">
            <input class="form-control" type="text" id="email" name="email">
          </div>
        </div>
        <div class="form-group">
          <label class="control-label col-sm-3" for="password">password</label>
          <div class="col-sm-9">
            <input class="form-control" type="password" id="password" name="password">
          </div>
        </div>
        <div class="col-sm-9 col-sm-offset-3">
          <input type="submit" class="btn btn-primary" value="Login">
          <a class="btn btn-link" href="<?=url('/login/password_reminder')?>">forget password</a>
        </div>
      </form>
      <?php endif ?>

      <?php if($enable_google_auth): ?>
      <div class="google-login col-sm-10 col-sm-offset-1">
        <a class="btn btn-primary col-xs-12" href="<?=url('/login/google')?>">Login with google account</a>
      </div>
      <?php endif ?>
    </div>
  </div>
</div>
