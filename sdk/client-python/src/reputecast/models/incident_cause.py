from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="IncidentCause")



@_attrs_define
class IncidentCause:
    """ 
        Attributes:
            body (str):
            lead (str):
     """

    body: str
    lead: str





    def to_dict(self) -> dict[str, Any]:
        body = self.body

        lead = self.lead


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "body": body,
            "lead": lead,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        body = d.pop("body")

        lead = d.pop("lead")

        incident_cause = cls(
            body=body,
            lead=lead,
        )

        return incident_cause

