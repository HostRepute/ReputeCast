from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="ReplyAuth")



@_attrs_define
class ReplyAuth:
    """ 
        Attributes:
            dkim (str | Unset):
            dmarc (str | Unset):
            spf (str | Unset):
     """

    dkim: str | Unset = UNSET
    dmarc: str | Unset = UNSET
    spf: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        dkim = self.dkim

        dmarc = self.dmarc

        spf = self.spf


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if dkim is not UNSET:
            field_dict["dkim"] = dkim
        if dmarc is not UNSET:
            field_dict["dmarc"] = dmarc
        if spf is not UNSET:
            field_dict["spf"] = spf

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        dkim = d.pop("dkim", UNSET)

        dmarc = d.pop("dmarc", UNSET)

        spf = d.pop("spf", UNSET)

        reply_auth = cls(
            dkim=dkim,
            dmarc=dmarc,
            spf=spf,
        )

        return reply_auth

