from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.signup_form_targeting_devices_item import SignupFormTargetingDevicesItem
from ..models.signup_form_targeting_visitors import SignupFormTargetingVisitors
from typing import cast

if TYPE_CHECKING:
  from ..models.signup_form_countries import SignupFormCountries
  from ..models.signup_form_page_rules import SignupFormPageRules
  from ..models.signup_form_schedule import SignupFormSchedule
  from ..models.signup_form_sources import SignupFormSources





T = TypeVar("T", bound="SignupFormTargeting")



@_attrs_define
class SignupFormTargeting:
    """ 
        Attributes:
            countries (SignupFormCountries):
            devices (list[SignupFormTargetingDevicesItem]): None: every device
            pages (SignupFormPageRules):
            schedule (SignupFormSchedule):
            sources (SignupFormSources):
            visitors (SignupFormTargetingVisitors): Returning: this browser has seen one of the workspace's forms before
                today
     """

    countries: SignupFormCountries
    devices: list[SignupFormTargetingDevicesItem]
    pages: SignupFormPageRules
    schedule: SignupFormSchedule
    sources: SignupFormSources
    visitors: SignupFormTargetingVisitors





    def to_dict(self) -> dict[str, Any]:
        from ..models.signup_form_countries import SignupFormCountries # noqa: PLC0415
        from ..models.signup_form_page_rules import SignupFormPageRules # noqa: PLC0415
        from ..models.signup_form_schedule import SignupFormSchedule # noqa: PLC0415
        from ..models.signup_form_sources import SignupFormSources # noqa: PLC0415
        countries = self.countries.to_dict()

        devices = []
        for devices_item_data in self.devices:
            devices_item = devices_item_data.value
            devices.append(devices_item)



        pages = self.pages.to_dict()

        schedule = self.schedule.to_dict()

        sources = self.sources.to_dict()

        visitors = self.visitors.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "countries": countries,
            "devices": devices,
            "pages": pages,
            "schedule": schedule,
            "sources": sources,
            "visitors": visitors,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.signup_form_countries import SignupFormCountries # noqa: PLC0415
        from ..models.signup_form_page_rules import SignupFormPageRules # noqa: PLC0415
        from ..models.signup_form_schedule import SignupFormSchedule # noqa: PLC0415
        from ..models.signup_form_sources import SignupFormSources # noqa: PLC0415
        d = dict(src_dict)
        countries = SignupFormCountries.from_dict(d.pop("countries"))




        devices = []
        _devices = d.pop("devices")
        for devices_item_data in (_devices):
            devices_item = SignupFormTargetingDevicesItem(devices_item_data)



            devices.append(devices_item)


        pages = SignupFormPageRules.from_dict(d.pop("pages"))




        schedule = SignupFormSchedule.from_dict(d.pop("schedule"))




        sources = SignupFormSources.from_dict(d.pop("sources"))




        visitors = SignupFormTargetingVisitors(d.pop("visitors"))




        signup_form_targeting = cls(
            countries=countries,
            devices=devices,
            pages=pages,
            schedule=schedule,
            sources=sources,
            visitors=visitors,
        )

        return signup_form_targeting

