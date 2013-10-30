
<div class="well">
  <form class="form-horizontal">
    <fieldset>
      <legend>New application</legend>

      <div class="row">
        <div class="col-md-10 col-sm-9">
          <div class="form-group">
            <label for="title" class="control-label col-md-2">Title</label>
            <div class="col-md-10">
              <input class="form-control" type="text" id="title" name="title">
            </div>
          </div>
        
          <div class="form-group">
            <label for="icon" class="control-label col-md-2">Icon</label>
            <input type="file" id="icon" class="hidden">
            <div class="input-group col-md-10">
              <input type="text" class="form-control">
              <a class="input-group-addon btn btn-default">Browse</a>
            </div>
          </div>
        </div>
      
        <div class="col-md-2 col-sm-3 hidden-xs text-center">
          <img class="img-thumbnail" style="width:96px;height:96px">
        </div>
      </div>
      
      <div class="row">
        <div class="form-group">
          <div class="col-md-10">
            <label for="description" class="control-label col-md-2">Description</label>
            <div class="col-md-10">
              <textarea class="form-control" row="3" id="description" name="description"></textarea>
            </div>
          </div>
        </div>
      </div>

      <div class="row">
        <div class="form-group">
          <div class="col-md-10">
            <div class="col-md-10 col-md-offset-2">
              <input type="submit" class="btn btn-primary" value="Create">
            </div>
          </div>
        </div>
      </div>

    </fieldset>
  </form>

</div>

