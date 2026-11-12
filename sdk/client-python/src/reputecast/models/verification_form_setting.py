from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="VerificationFormSetting")



@_attrs_define
class VerificationFormSetting:
    """ 
        Attributes:
            asked_to_fix (int):
            can_change (bool):
            down (bool):
            enabled (bool):
            form_key (str):
            helpers (list[str]):
            price (float):
            waiting (int):
            service (str | Unset): The forms default service; empty when there is none
            service_name (str | Unset):
     """

    asked_to_fix: int
    can_change: bool
    down: bool
    enabled: bool
    form_key: str
    helpers: list[str]
    price: float
    waiting: int
    service: str | Unset = UNSET
    service_name: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        asked_to_fix = self.asked_to_fix

        can_change = self.can_change

        down = self.down

        enabled = self.enabled

        form_key = self.form_key

        helpers = self.helpers



        price = self.price

        waiting = self.waiting

        service = self.service

        service_name = self.service_name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "askedToFix": asked_to_fix,
            "canChange": can_change,
            "down": down,
            "enabled": enabled,
            "formKey": form_key,
            "helpers": helpers,
            "price": price,
            "waiting": waiting,
        })
        if service is not UNSET:
            field_dict["service"] = service
        if service_name is not UNSET:
            field_dict["serviceName"] = service_name

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        asked_to_fix = d.pop("askedToFix")

        can_change = d.pop("canChange")

        down = d.pop("down")

        enabled = d.pop("enabled")

        form_key = d.pop("formKey")

        helpers = cast(list[str], d.pop("helpers"))


        price = d.pop("price")

        waiting = d.pop("waiting")

        service = d.pop("service", UNSET)

        service_name = d.pop("serviceName", UNSET)

        verification_form_setting = cls(
            asked_to_fix=asked_to_fix,
            can_change=can_change,
            down=down,
            enabled=enabled,
            form_key=form_key,
            helpers=helpers,
            price=price,
            waiting=waiting,
            service=service,
            service_name=service_name,
        )

        return verification_form_setting

