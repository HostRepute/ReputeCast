from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.growth_source import GrowthSource





T = TypeVar("T", bound="ContactJoin")



@_attrs_define
class ContactJoin:
    """ 
        Attributes:
            attributed (bool):
            page (str):
            referrer (str):
            source (GrowthSource):
            utm_campaign (str):
            utm_medium (str):
            utm_source (str):
            joined_at (datetime.datetime | Unset):
            left_at (datetime.datetime | Unset):
            left_kind (str | Unset):
     """

    attributed: bool
    page: str
    referrer: str
    source: GrowthSource
    utm_campaign: str
    utm_medium: str
    utm_source: str
    joined_at: datetime.datetime | Unset = UNSET
    left_at: datetime.datetime | Unset = UNSET
    left_kind: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.growth_source import GrowthSource # noqa: PLC0415
        attributed = self.attributed

        page = self.page

        referrer = self.referrer

        source = self.source.to_dict()

        utm_campaign = self.utm_campaign

        utm_medium = self.utm_medium

        utm_source = self.utm_source

        joined_at: str | Unset = UNSET
        if not isinstance(self.joined_at, Unset):
            joined_at = self.joined_at.isoformat()

        left_at: str | Unset = UNSET
        if not isinstance(self.left_at, Unset):
            left_at = self.left_at.isoformat()

        left_kind = self.left_kind


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "attributed": attributed,
            "page": page,
            "referrer": referrer,
            "source": source,
            "utmCampaign": utm_campaign,
            "utmMedium": utm_medium,
            "utmSource": utm_source,
        })
        if joined_at is not UNSET:
            field_dict["joinedAt"] = joined_at
        if left_at is not UNSET:
            field_dict["leftAt"] = left_at
        if left_kind is not UNSET:
            field_dict["leftKind"] = left_kind

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.growth_source import GrowthSource # noqa: PLC0415
        d = dict(src_dict)
        attributed = d.pop("attributed")

        page = d.pop("page")

        referrer = d.pop("referrer")

        source = GrowthSource.from_dict(d.pop("source"))




        utm_campaign = d.pop("utmCampaign")

        utm_medium = d.pop("utmMedium")

        utm_source = d.pop("utmSource")

        _joined_at = d.pop("joinedAt", UNSET)
        joined_at: datetime.datetime | Unset
        if isinstance(_joined_at,  Unset):
            joined_at = UNSET
        else:
            joined_at = datetime.datetime.fromisoformat(_joined_at)




        _left_at = d.pop("leftAt", UNSET)
        left_at: datetime.datetime | Unset
        if isinstance(_left_at,  Unset):
            left_at = UNSET
        else:
            left_at = datetime.datetime.fromisoformat(_left_at)




        left_kind = d.pop("leftKind", UNSET)

        contact_join = cls(
            attributed=attributed,
            page=page,
            referrer=referrer,
            source=source,
            utm_campaign=utm_campaign,
            utm_medium=utm_medium,
            utm_source=utm_source,
            joined_at=joined_at,
            left_at=left_at,
            left_kind=left_kind,
        )

        return contact_join

