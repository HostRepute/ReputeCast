from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="CanaryGates")



@_attrs_define
class CanaryGates:
    """ 
        Attributes:
            block_bounce (float):
            bounce (float):
            complaint (float):
            deferral (float):
            seed_spam (int):
     """

    block_bounce: float
    bounce: float
    complaint: float
    deferral: float
    seed_spam: int





    def to_dict(self) -> dict[str, Any]:
        block_bounce = self.block_bounce

        bounce = self.bounce

        complaint = self.complaint

        deferral = self.deferral

        seed_spam = self.seed_spam


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "blockBounce": block_bounce,
            "bounce": bounce,
            "complaint": complaint,
            "deferral": deferral,
            "seedSpam": seed_spam,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        block_bounce = d.pop("blockBounce")

        bounce = d.pop("bounce")

        complaint = d.pop("complaint")

        deferral = d.pop("deferral")

        seed_spam = d.pop("seedSpam")

        canary_gates = cls(
            block_bounce=block_bounce,
            bounce=bounce,
            complaint=complaint,
            deferral=deferral,
            seed_spam=seed_spam,
        )

        return canary_gates

