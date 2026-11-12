from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from uuid import UUID






T = TypeVar("T", bound="LiveAcknowledgeRequest")



@_attrs_define
class LiveAcknowledgeRequest:
    """ 
        Attributes:
            incident_id (UUID):
     """

    incident_id: UUID





    def to_dict(self) -> dict[str, Any]:
        incident_id = str(self.incident_id)


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "incidentId": incident_id,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        incident_id = UUID(d.pop("incidentId"))




        live_acknowledge_request = cls(
            incident_id=incident_id,
        )

        return live_acknowledge_request

