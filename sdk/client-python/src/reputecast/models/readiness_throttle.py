from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.readiness_throttle_key import ReadinessThrottleKey






T = TypeVar("T", bound="ReadinessThrottle")



@_attrs_define
class ReadinessThrottle:
    """ 
        Attributes:
            key (ReadinessThrottleKey):
            name (str):
            rate (float):
            recipients (int):
     """

    key: ReadinessThrottleKey
    name: str
    rate: float
    recipients: int





    def to_dict(self) -> dict[str, Any]:
        key = self.key.value

        name = self.name

        rate = self.rate

        recipients = self.recipients


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key": key,
            "name": name,
            "rate": rate,
            "recipients": recipients,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        key = ReadinessThrottleKey(d.pop("key"))




        name = d.pop("name")

        rate = d.pop("rate")

        recipients = d.pop("recipients")

        readiness_throttle = cls(
            key=key,
            name=name,
            rate=rate,
            recipients=recipients,
        )

        return readiness_throttle

