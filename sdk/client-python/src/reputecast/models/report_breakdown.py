from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.report_dim_row import ReportDimRow
  from ..models.report_time_bucket import ReportTimeBucket





T = TypeVar("T", bound="ReportBreakdown")



@_attrs_define
class ReportBreakdown:
    """ 
        Attributes:
            cities (list[ReportDimRow]):
            clients (list[ReportDimRow]):
            countries (list[ReportDimRow]):
            devices (list[ReportDimRow]):
            geoip (bool):
            hours (list[ReportTimeBucket]):
            located (int):
            measured (bool):
            privacy_opens (int):
            systems (list[ReportDimRow]):
            timezone (str):
            weekdays (list[ReportTimeBucket]):
     """

    cities: list[ReportDimRow]
    clients: list[ReportDimRow]
    countries: list[ReportDimRow]
    devices: list[ReportDimRow]
    geoip: bool
    hours: list[ReportTimeBucket]
    located: int
    measured: bool
    privacy_opens: int
    systems: list[ReportDimRow]
    timezone: str
    weekdays: list[ReportTimeBucket]





    def to_dict(self) -> dict[str, Any]:
        from ..models.report_dim_row import ReportDimRow # noqa: PLC0415
        from ..models.report_time_bucket import ReportTimeBucket # noqa: PLC0415
        cities = []
        for cities_item_data in self.cities:
            cities_item = cities_item_data.to_dict()
            cities.append(cities_item)



        clients = []
        for clients_item_data in self.clients:
            clients_item = clients_item_data.to_dict()
            clients.append(clients_item)



        countries = []
        for countries_item_data in self.countries:
            countries_item = countries_item_data.to_dict()
            countries.append(countries_item)



        devices = []
        for devices_item_data in self.devices:
            devices_item = devices_item_data.to_dict()
            devices.append(devices_item)



        geoip = self.geoip

        hours = []
        for hours_item_data in self.hours:
            hours_item = hours_item_data.to_dict()
            hours.append(hours_item)



        located = self.located

        measured = self.measured

        privacy_opens = self.privacy_opens

        systems = []
        for systems_item_data in self.systems:
            systems_item = systems_item_data.to_dict()
            systems.append(systems_item)



        timezone = self.timezone

        weekdays = []
        for weekdays_item_data in self.weekdays:
            weekdays_item = weekdays_item_data.to_dict()
            weekdays.append(weekdays_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "cities": cities,
            "clients": clients,
            "countries": countries,
            "devices": devices,
            "geoip": geoip,
            "hours": hours,
            "located": located,
            "measured": measured,
            "privacyOpens": privacy_opens,
            "systems": systems,
            "timezone": timezone,
            "weekdays": weekdays,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.report_dim_row import ReportDimRow # noqa: PLC0415
        from ..models.report_time_bucket import ReportTimeBucket # noqa: PLC0415
        d = dict(src_dict)
        cities = []
        _cities = d.pop("cities")
        for cities_item_data in (_cities):
            cities_item = ReportDimRow.from_dict(cities_item_data)



            cities.append(cities_item)


        clients = []
        _clients = d.pop("clients")
        for clients_item_data in (_clients):
            clients_item = ReportDimRow.from_dict(clients_item_data)



            clients.append(clients_item)


        countries = []
        _countries = d.pop("countries")
        for countries_item_data in (_countries):
            countries_item = ReportDimRow.from_dict(countries_item_data)



            countries.append(countries_item)


        devices = []
        _devices = d.pop("devices")
        for devices_item_data in (_devices):
            devices_item = ReportDimRow.from_dict(devices_item_data)



            devices.append(devices_item)


        geoip = d.pop("geoip")

        hours = []
        _hours = d.pop("hours")
        for hours_item_data in (_hours):
            hours_item = ReportTimeBucket.from_dict(hours_item_data)



            hours.append(hours_item)


        located = d.pop("located")

        measured = d.pop("measured")

        privacy_opens = d.pop("privacyOpens")

        systems = []
        _systems = d.pop("systems")
        for systems_item_data in (_systems):
            systems_item = ReportDimRow.from_dict(systems_item_data)



            systems.append(systems_item)


        timezone = d.pop("timezone")

        weekdays = []
        _weekdays = d.pop("weekdays")
        for weekdays_item_data in (_weekdays):
            weekdays_item = ReportTimeBucket.from_dict(weekdays_item_data)



            weekdays.append(weekdays_item)


        report_breakdown = cls(
            cities=cities,
            clients=clients,
            countries=countries,
            devices=devices,
            geoip=geoip,
            hours=hours,
            located=located,
            measured=measured,
            privacy_opens=privacy_opens,
            systems=systems,
            timezone=timezone,
            weekdays=weekdays,
        )

        return report_breakdown

