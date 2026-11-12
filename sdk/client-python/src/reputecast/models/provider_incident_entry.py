from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.provider_incident_entry_severity import ProviderIncidentEntrySeverity
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ProviderIncidentEntry")



@_attrs_define
class ProviderIncidentEntry:
    """ 
        Attributes:
            at (datetime.datetime):
            detail (str):
            id (str):
            kind (str):
            severity (ProviderIncidentEntrySeverity):
            title (str):
            campaign_id (str | Unset):
     """

    at: datetime.datetime
    detail: str
    id: str
    kind: str
    severity: ProviderIncidentEntrySeverity
    title: str
    campaign_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        detail = self.detail

        id = self.id

        kind = self.kind

        severity = self.severity.value

        title = self.title

        campaign_id = self.campaign_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "detail": detail,
            "id": id,
            "kind": kind,
            "severity": severity,
            "title": title,
        })
        if campaign_id is not UNSET:
            field_dict["campaignId"] = campaign_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        detail = d.pop("detail")

        id = d.pop("id")

        kind = d.pop("kind")

        severity = ProviderIncidentEntrySeverity(d.pop("severity"))




        title = d.pop("title")

        campaign_id = d.pop("campaignId", UNSET)

        provider_incident_entry = cls(
            at=at,
            detail=detail,
            id=id,
            kind=kind,
            severity=severity,
            title=title,
            campaign_id=campaign_id,
        )

        return provider_incident_entry

