from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="SmsThresholds")



@_attrs_define
class SmsThresholds:
    """ 
        Attributes:
            failed (float):
            filtered (float):
            opt_outs (float):
     """

    failed: float
    filtered: float
    opt_outs: float





    def to_dict(self) -> dict[str, Any]:
        failed = self.failed

        filtered = self.filtered

        opt_outs = self.opt_outs


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "failed": failed,
            "filtered": filtered,
            "optOuts": opt_outs,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        failed = d.pop("failed")

        filtered = d.pop("filtered")

        opt_outs = d.pop("optOuts")

        sms_thresholds = cls(
            failed=failed,
            filtered=filtered,
            opt_outs=opt_outs,
        )

        return sms_thresholds

