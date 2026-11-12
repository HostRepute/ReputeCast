from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.deliverability_delisting_outcome_status import DeliverabilityDelistingOutcomeStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="DeliverabilityDelistingOutcome")



@_attrs_define
class DeliverabilityDelistingOutcome:
    """ 
        Attributes:
            already (bool):
            requested_at (datetime.datetime):
            source (str):
            status (DeliverabilityDelistingOutcomeStatus):
            url (str | Unset):
     """

    already: bool
    requested_at: datetime.datetime
    source: str
    status: DeliverabilityDelistingOutcomeStatus
    url: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        already = self.already

        requested_at = self.requested_at.isoformat()

        source = self.source

        status = self.status.value

        url = self.url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "already": already,
            "requestedAt": requested_at,
            "source": source,
            "status": status,
        })
        if url is not UNSET:
            field_dict["url"] = url

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        already = d.pop("already")

        requested_at = datetime.datetime.fromisoformat(d.pop("requestedAt"))




        source = d.pop("source")

        status = DeliverabilityDelistingOutcomeStatus(d.pop("status"))




        url = d.pop("url", UNSET)

        deliverability_delisting_outcome = cls(
            already=already,
            requested_at=requested_at,
            source=source,
            status=status,
            url=url,
        )

        return deliverability_delisting_outcome

