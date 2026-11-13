from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.review_campaign_status import ReviewCampaignStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.composer_content_settings import ComposerContentSettings
  from ..models.review_ab import ReviewAB
  from ..models.review_holdout import ReviewHoldout
  from ..models.review_utm import ReviewUTM





T = TypeVar("T", bound="ReviewCampaign")



@_attrs_define
class ReviewCampaign:
    """ 
        Attributes:
            audience_label (str):
            editable (bool):
            from_ (str):
            id (str):
            name (str):
            preheader (str):
            reply_to (str):
            settings (ComposerContentSettings):
            status (ReviewCampaignStatus):
            subject (str):
            updated_at (datetime.datetime):
            updated_by (str):
            utm_default (bool):
            version (int):
            ab (ReviewAB | Unset):
            autosaved_at (datetime.datetime | Unset):
            holdout (ReviewHoldout | Unset):
            scheduled_at (datetime.datetime | Unset):
            started_at (datetime.datetime | Unset):
            utm (ReviewUTM | Unset):
     """

    audience_label: str
    editable: bool
    from_: str
    id: str
    name: str
    preheader: str
    reply_to: str
    settings: ComposerContentSettings
    status: ReviewCampaignStatus
    subject: str
    updated_at: datetime.datetime
    updated_by: str
    utm_default: bool
    version: int
    ab: ReviewAB | Unset = UNSET
    autosaved_at: datetime.datetime | Unset = UNSET
    holdout: ReviewHoldout | Unset = UNSET
    scheduled_at: datetime.datetime | Unset = UNSET
    started_at: datetime.datetime | Unset = UNSET
    utm: ReviewUTM | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.composer_content_settings import ComposerContentSettings # noqa: PLC0415
        from ..models.review_ab import ReviewAB # noqa: PLC0415
        from ..models.review_holdout import ReviewHoldout # noqa: PLC0415
        from ..models.review_utm import ReviewUTM # noqa: PLC0415
        audience_label = self.audience_label

        editable = self.editable

        from_ = self.from_

        id = self.id

        name = self.name

        preheader = self.preheader

        reply_to = self.reply_to

        settings = self.settings.to_dict()

        status = self.status.value

        subject = self.subject

        updated_at = self.updated_at.isoformat()

        updated_by = self.updated_by

        utm_default = self.utm_default

        version = self.version

        ab: dict[str, Any] | Unset = UNSET
        if not isinstance(self.ab, Unset):
            ab = self.ab.to_dict()

        autosaved_at: str | Unset = UNSET
        if not isinstance(self.autosaved_at, Unset):
            autosaved_at = self.autosaved_at.isoformat()

        holdout: dict[str, Any] | Unset = UNSET
        if not isinstance(self.holdout, Unset):
            holdout = self.holdout.to_dict()

        scheduled_at: str | Unset = UNSET
        if not isinstance(self.scheduled_at, Unset):
            scheduled_at = self.scheduled_at.isoformat()

        started_at: str | Unset = UNSET
        if not isinstance(self.started_at, Unset):
            started_at = self.started_at.isoformat()

        utm: dict[str, Any] | Unset = UNSET
        if not isinstance(self.utm, Unset):
            utm = self.utm.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "audienceLabel": audience_label,
            "editable": editable,
            "from": from_,
            "id": id,
            "name": name,
            "preheader": preheader,
            "replyTo": reply_to,
            "settings": settings,
            "status": status,
            "subject": subject,
            "updatedAt": updated_at,
            "updatedBy": updated_by,
            "utmDefault": utm_default,
            "version": version,
        })
        if ab is not UNSET:
            field_dict["ab"] = ab
        if autosaved_at is not UNSET:
            field_dict["autosavedAt"] = autosaved_at
        if holdout is not UNSET:
            field_dict["holdout"] = holdout
        if scheduled_at is not UNSET:
            field_dict["scheduledAt"] = scheduled_at
        if started_at is not UNSET:
            field_dict["startedAt"] = started_at
        if utm is not UNSET:
            field_dict["utm"] = utm

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.composer_content_settings import ComposerContentSettings # noqa: PLC0415
        from ..models.review_ab import ReviewAB # noqa: PLC0415
        from ..models.review_holdout import ReviewHoldout # noqa: PLC0415
        from ..models.review_utm import ReviewUTM # noqa: PLC0415
        d = dict(src_dict)
        audience_label = d.pop("audienceLabel")

        editable = d.pop("editable")

        from_ = d.pop("from")

        id = d.pop("id")

        name = d.pop("name")

        preheader = d.pop("preheader")

        reply_to = d.pop("replyTo")

        settings = ComposerContentSettings.from_dict(d.pop("settings"))




        status = ReviewCampaignStatus(d.pop("status"))




        subject = d.pop("subject")

        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))




        updated_by = d.pop("updatedBy")

        utm_default = d.pop("utmDefault")

        version = d.pop("version")

        _ab = d.pop("ab", UNSET)
        ab: ReviewAB | Unset
        if isinstance(_ab,  Unset):
            ab = UNSET
        else:
            ab = ReviewAB.from_dict(_ab)




        _autosaved_at = d.pop("autosavedAt", UNSET)
        autosaved_at: datetime.datetime | Unset
        if isinstance(_autosaved_at,  Unset):
            autosaved_at = UNSET
        else:
            autosaved_at = datetime.datetime.fromisoformat(_autosaved_at)




        _holdout = d.pop("holdout", UNSET)
        holdout: ReviewHoldout | Unset
        if isinstance(_holdout,  Unset):
            holdout = UNSET
        else:
            holdout = ReviewHoldout.from_dict(_holdout)




        _scheduled_at = d.pop("scheduledAt", UNSET)
        scheduled_at: datetime.datetime | Unset
        if isinstance(_scheduled_at,  Unset):
            scheduled_at = UNSET
        else:
            scheduled_at = datetime.datetime.fromisoformat(_scheduled_at)




        _started_at = d.pop("startedAt", UNSET)
        started_at: datetime.datetime | Unset
        if isinstance(_started_at,  Unset):
            started_at = UNSET
        else:
            started_at = datetime.datetime.fromisoformat(_started_at)




        _utm = d.pop("utm", UNSET)
        utm: ReviewUTM | Unset
        if isinstance(_utm,  Unset):
            utm = UNSET
        else:
            utm = ReviewUTM.from_dict(_utm)




        review_campaign = cls(
            audience_label=audience_label,
            editable=editable,
            from_=from_,
            id=id,
            name=name,
            preheader=preheader,
            reply_to=reply_to,
            settings=settings,
            status=status,
            subject=subject,
            updated_at=updated_at,
            updated_by=updated_by,
            utm_default=utm_default,
            version=version,
            ab=ab,
            autosaved_at=autosaved_at,
            holdout=holdout,
            scheduled_at=scheduled_at,
            started_at=started_at,
            utm=utm,
        )

        return review_campaign

