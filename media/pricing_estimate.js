
const babyLimitDate = new Date(2021, 8, 30);
const kidLimitDate = new Date(2013, 8, 30);


function computeCost() {
  var rsvps = $('select[name="rsvp"]').map(function() {return $(this).val()}).toArray();
  var totalAttendees = 0;
  for (var x = 0; x < rsvps.length; x++) {
    if (rsvps[x] == "2" || rsvps[x] == "4" || rsvps[x] =="5") totalAttendees++;
  }
  
  if (totalAttendees == 1 && $('select[name="housingPreference"]').val() != "1") {
    $(".cost0Roommates").text("$" + computeCostWithRoommates(0));
    $(".cost1Roommates").text("$" + computeCostWithRoommates(1));
    $(".cost2Roommates").text("$" + computeCostWithRoommates(2));
    $(".cost3Roommates").text("$" + computeCostWithRoommates(3)); 
    $(".onlyCost").hide();
    $(".roommateCosts").show();

  } else {
    $(".onlyCost").text("$" + computeCostWithRoommates(0));
    $(".roommateCosts").hide();
    $(".onlyCost").show();
  }
}

function computeCostWithRoommates(additionalPeople) {

  var numberOfNights = 0;
  if (anyoneMeetsCriteria(attending)) {
    numberOfNights = 2;
  }
  if (anyoneMeetsCriteria(threeNights)) {
    numberOfNights = 3;
  }

    // Room rate:  Carrigain 290  O/T 255  KP 210
    // If only one cost class is selected use that
    // If more than one selected, use O/T if selected.
    // If CR and KP selected choose KP

  var roomRate = 255;
  var roomRateDescription = "Default room rate of $255 per night";
    // this was prefer more expensive room I think
    //if ($('input[name="housingPreferenceBooleans"][value="7"]').is(":checked")) roomRate = 390;
  if ($('input[name="housingPreferenceBooleans"][value="1"]').is(":checked")) {
    roomRate = 210;
    roomRateDescription = "Selected King Pine room rate of $210 per night";
  }
  if ($('input[name="housingPreferenceBooleans"][value="0"]').is(":checked")) {
    roomRate = 255;
    roomRateDescription = "Selected Osceola/Tecumseh room rate of $255 per night";
  }
  if ($('input[name="housingPreferenceBooleans"][value="2"]').is(":checked")) {
    roomRate = 290;
    roomRateDescription = "Selected Carrigain room rate of $290 per night";
  }

  var rsvps = $('select[name="rsvp"]').map(function() {return $(this).val()}).toArray();
  var totalAttendees = 0;
  for (var x = 0; x < rsvps.length; x++) {
    if (rsvps[x] == "2" || rsvps[x] == "4" || rsvps[x] == "5") totalAttendees++;
  }

  // if only 1 person and no roommates, set room rate to single room cost
  if (totalAttendees == 1) {
    if ($('select[name="housingPreference"]').val() != "1") {
      roomRate /= (additionalPeople + 1); // this is correct because we only get here if this is a single person
    }
  }


  var lodgingCost = roomRate * numberOfNights;
  //console.log("lodging cost: " + lodgingCost);
  var foodCost = 0;


  var adultFridayDinnerCost = 15;
  var kidFridayDinnerCost = 10;
  var adultMealCost = 50;
  var adultCateredCost = 10;    
  var kidMealCost = 25;
  var kidCateredCost = 5;
  // var babyFoodCost = 0;
  
  const pricesByAgeAndNights = [
      [adultFridayDinnerCost + adultMealCost * 3 + adultCateredCost * 2, adultFridayDinnerCost + adultMealCost * 2 + adultCateredCost * 2],
      [kidFridayDinnerCost + kidMealCost * 3 + kidCateredCost * 2, kidFridayDinnerCost + kidMealCost * 2 + kidCateredCost * 2],
      [0, 0]
  ]

  var birthdates = $('input[name="Birthdate"]').map(function() {return $(this).val()}).toArray();

  for (var i=0; i < rsvps.length; i++) {

    var nightsIndex = -1;
    if (rsvps[i] == "5") nightsIndex = 0;
    else if (rsvps[i] == "2") nightsIndex = 1;
//    else if (rsvps[i] == "4") nightsIndex = 2;
    else continue;

    var ageIndex = 0;
    const birthdateStr = birthdates[i];
    if (birthdateStr && birthdateStr.split("/").length == 3) {
      var pieces = birthdateStr.split("/").map(function(x) {return Number(x);})

      const date = new Date(pieces[2], pieces[0], pieces[1]);
      if (date > babyLimitDate) ageIndex = 2;
      else if (date > kidLimitDate) ageIndex = 1;
    }
    
    foodCost += pricesByAgeAndNights[ageIndex][nightsIndex];
  }

  //console.log("food cost: " + foodCost); 

    var incidentalsPerPersonPerNight = 5; // assumes $800 of expenses, 50 people staying 3 nights, 20 people staying 2 nights, no covid tests
 
  var incidentalsCost = incidentalsPerPersonPerNight *  $('select[name="rsvp"] option:selected').map(function() {
    if ($( this ).val() == "5") return 3;
    if ($( this ).val() == "2") return 2;
    if ($( this ).val() == "4") return 2;
  }).toArray().reduce((current, total) => current + total, 0)
      //console.log("incidentals cost: " + incidentalsCost);

  var totalCost = lodgingCost + foodCost + incidentalsCost;
  $(".estimatedCost").text("$" + totalCost + " (" + roomRateDescription + ")");

      //console.log(totalCost);
  //alert(totalCost);
  return Math.ceil(totalCost);

}

