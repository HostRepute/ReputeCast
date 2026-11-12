from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="SmsClickLink")



@_attrs_define
class SmsClickLink:
    """ 
        Attributes:
            domain (str):
            machine (int):
            people (int):
            url (str):
     """

    domain: str
    machine: int
    people: int
    url: str





    def to_dict(self) -> dict[str, Any]:
        domain = self.domain

        machine = self.machine

        people = self.people

        url = self.url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "domain": domain,
            "machine": machine,
            "people": people,
            "url": url,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        domain = d.pop("domain")

        machine = d.pop("machine")

        people = d.pop("people")

        url = d.pop("url")

        sms_click_link = cls(
            domain=domain,
            machine=machine,
            people=people,
            url=url,
        )

        return sms_click_link

