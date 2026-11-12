from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="ReportInput")



@_attrs_define
class ReportInput:
    """ 
        Attributes:
            html (str | Unset):
            mjml (str | Unset):
            recount (bool | Unset):
     """

    html: str | Unset = UNSET
    mjml: str | Unset = UNSET
    recount: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        html = self.html

        mjml = self.mjml

        recount = self.recount


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if html is not UNSET:
            field_dict["html"] = html
        if mjml is not UNSET:
            field_dict["mjml"] = mjml
        if recount is not UNSET:
            field_dict["recount"] = recount

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        html = d.pop("html", UNSET)

        mjml = d.pop("mjml", UNSET)

        recount = d.pop("recount", UNSET)

        report_input = cls(
            html=html,
            mjml=mjml,
            recount=recount,
        )

        return report_input

