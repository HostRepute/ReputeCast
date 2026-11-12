from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.deliverability_incident_severity import DeliverabilityIncidentSeverity
from ..models.deliverability_incident_source import DeliverabilityIncidentSource
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.deliverability_incident_link import DeliverabilityIncidentLink





T = TypeVar("T", bound="DeliverabilityIncident")



@_attrs_define
class DeliverabilityIncident:
    """ 
        Attributes:
            detail (str):
            id (str):
            kind (str):
            links (list[DeliverabilityIncidentLink]):
            occurred_at (datetime.datetime):
            severity (DeliverabilityIncidentSeverity):
            source (DeliverabilityIncidentSource):
            title (str):
     """

    detail: str
    id: str
    kind: str
    links: list[DeliverabilityIncidentLink]
    occurred_at: datetime.datetime
    severity: DeliverabilityIncidentSeverity
    source: DeliverabilityIncidentSource
    title: str





    def to_dict(self) -> dict[str, Any]:
        from ..models.deliverability_incident_link import DeliverabilityIncidentLink # noqa: PLC0415
        detail = self.detail

        id = self.id

        kind = self.kind

        links = []
        for links_item_data in self.links:
            links_item = links_item_data.to_dict()
            links.append(links_item)



        occurred_at = self.occurred_at.isoformat()

        severity = self.severity.value

        source = self.source.value

        title = self.title


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "detail": detail,
            "id": id,
            "kind": kind,
            "links": links,
            "occurredAt": occurred_at,
            "severity": severity,
            "source": source,
            "title": title,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.deliverability_incident_link import DeliverabilityIncidentLink # noqa: PLC0415
        d = dict(src_dict)
        detail = d.pop("detail")

        id = d.pop("id")

        kind = d.pop("kind")

        links = []
        _links = d.pop("links")
        for links_item_data in (_links):
            links_item = DeliverabilityIncidentLink.from_dict(links_item_data)



            links.append(links_item)


        occurred_at = datetime.datetime.fromisoformat(d.pop("occurredAt"))




        severity = DeliverabilityIncidentSeverity(d.pop("severity"))




        source = DeliverabilityIncidentSource(d.pop("source"))




        title = d.pop("title")

        deliverability_incident = cls(
            detail=detail,
            id=id,
            kind=kind,
            links=links,
            occurred_at=occurred_at,
            severity=severity,
            source=source,
            title=title,
        )

        return deliverability_incident

