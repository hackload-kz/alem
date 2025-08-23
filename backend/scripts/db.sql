select count(*) from seats;

select '----';
select 
    price,
    count(*)
from seats
group by price
;


select '----';
select 
    price,
    min((row - 1) * 1000 + number),
    max((row - 1) * 1000 + number)
from seats 
group by price
;

select '----';
select 
    *
from seats 
where row = 9 and number = 1000
;
