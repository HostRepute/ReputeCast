from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="DigestRuleMatch")



@_attrs_define
class DigestRuleMatch:
    """ 
        Attributes:
            key (str):
            label (str):
            short (str):
            threshold (int | Unset):
            until (datetime.datetime | Unset):
     """

    key: str
    label: str
    short: str
    threshold: int | Unset = UNSET
    until: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        key = self.key

        label = self.label

        short = self.short

        threshold = self.threshold

        until: str | Unset = UNSET
        if not isinstance(self.until, Unset):
            until = self.until.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key": key,
            "label": label,
            "short": short,
        })
        if threshold is not UNSET:
            field_dict["threshold"] = threshold
        if until is not UNSET:
            field_dict["until"] = until

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        key = d.pop("key")

        label = d.pop("label")

        short = d.pop("short")

        threshold = d.pop("threshold", UNSET)

        _until = d.pop("until", UNSET)
        until: datetime.datetime | Unset
        if isinstance(_until,  Unset):
            until = UNSET
        else:
            until = datetime.datetime.fromisoformat(_until)




        digest_rule_match = cls(
            key=key,
            label=label,
            short=short,
            threshold=threshold,
            until=until,
        )

        return digest_rule_match

