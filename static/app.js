mapboxgl.accessToken = 'pk.eyJ1IjoibmhvY2tpIiwiYSI6ImNqdGRmOWkxOTE2cDE0NHA5ZjFsNjQ2NmUifQ.WdBS1kF25GIRcDyDjDZBDg';
var map = new mapboxgl.Map({
  container: 'map',
  style: 'mapbox://styles/mapbox/streets-v11',
  center: [-75.5609589, 6.2597828],
  zoom: 10
});

function background(value) {
  if (value < 0) {
    return "black"
  } else if (value <= 12) {
    return "green"
  } else if (value <= 37) {
    return "#FCE75D"
  } else if (value <= 55) {
    return "#F88137"
  } else if (value <= 150) {
    return "#DC3135"
  }

  return "#53116A"
}

document.getElementById("update").innerHTML = data.generated_at;
map.addControl(new mapboxgl.NavigationControl());
map.on('load', function () {

  data.stations.forEach(element => {
    popupTemplate = `<h4>${element.name}</h4> <p>${element.description}</p>`;
    var popup = new mapboxgl.Popup({ offset: 25, className: 'my-class' })
      .setLngLat([element.longitude, element.latitude])
      .setHTML(popupTemplate)
      .addTo(map);

    var el = document.createElement('div');
    el.className = 'marker';
    el.innerText = element.value;
    el.style.background = background(element.value);
    if (element.value > 150 || element.value < 0) {
      el.style.color = '#FFF';
    }

    if (element.value < 0) {
      el.innerText = 'X';
    }

    new mapboxgl.Marker(el)
      .setLngLat([element.longitude, element.latitude])
      .setPopup(popup)
      .addTo(map)
  });

})
