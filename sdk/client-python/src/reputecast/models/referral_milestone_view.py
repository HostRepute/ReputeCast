from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.referral_milestone_view_type import ReferralMilestoneViewType
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.referral_file import ReferralFile
  from ..models.referral_image import ReferralImage
  from ..models.referral_milestone_email import ReferralMilestoneEmail
  from ..models.referral_milestone_health import ReferralMilestoneHealth
  from ..models.referral_milestone_stats import ReferralMilestoneStats





T = TypeVar("T", bound="ReferralMilestoneView")



@_attrs_define
class ReferralMilestoneView:
    """ 
        Attributes:
            ask_address (bool):
            codes_alert_below (int):
            codes_left (int):
            codes_uploaded (int):
            count (int):
            coupon_offer_id (str):
            coupon_offer_name (str):
            coupon_valid_days (int):
            download_url (str):
            email (ReferralMilestoneEmail):
            health (ReferralMilestoneHealth):
            id (str):
            reward_description (str):
            reward_name (str):
            stats (ReferralMilestoneStats):
            type_ (ReferralMilestoneViewType):
            webhook_secret_set (bool):
            webhook_url (str):
            file (ReferralFile | Unset):
            image (ReferralImage | Unset):
            webhook_called_at (datetime.datetime | Unset):
            webhook_status (int | Unset):
     """

    ask_address: bool
    codes_alert_below: int
    codes_left: int
    codes_uploaded: int
    count: int
    coupon_offer_id: str
    coupon_offer_name: str
    coupon_valid_days: int
    download_url: str
    email: ReferralMilestoneEmail
    health: ReferralMilestoneHealth
    id: str
    reward_description: str
    reward_name: str
    stats: ReferralMilestoneStats
    type_: ReferralMilestoneViewType
    webhook_secret_set: bool
    webhook_url: str
    file: ReferralFile | Unset = UNSET
    image: ReferralImage | Unset = UNSET
    webhook_called_at: datetime.datetime | Unset = UNSET
    webhook_status: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.referral_file import ReferralFile # noqa: PLC0415
        from ..models.referral_image import ReferralImage # noqa: PLC0415
        from ..models.referral_milestone_email import ReferralMilestoneEmail # noqa: PLC0415
        from ..models.referral_milestone_health import ReferralMilestoneHealth # noqa: PLC0415
        from ..models.referral_milestone_stats import ReferralMilestoneStats # noqa: PLC0415
        ask_address = self.ask_address

        codes_alert_below = self.codes_alert_below

        codes_left = self.codes_left

        codes_uploaded = self.codes_uploaded

        count = self.count

        coupon_offer_id = self.coupon_offer_id

        coupon_offer_name = self.coupon_offer_name

        coupon_valid_days = self.coupon_valid_days

        download_url = self.download_url

        email = self.email.to_dict()

        health = self.health.to_dict()

        id = self.id

        reward_description = self.reward_description

        reward_name = self.reward_name

        stats = self.stats.to_dict()

        type_ = self.type_.value

        webhook_secret_set = self.webhook_secret_set

        webhook_url = self.webhook_url

        file: dict[str, Any] | Unset = UNSET
        if not isinstance(self.file, Unset):
            file = self.file.to_dict()

        image: dict[str, Any] | Unset = UNSET
        if not isinstance(self.image, Unset):
            image = self.image.to_dict()

        webhook_called_at: str | Unset = UNSET
        if not isinstance(self.webhook_called_at, Unset):
            webhook_called_at = self.webhook_called_at.isoformat()

        webhook_status = self.webhook_status


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "askAddress": ask_address,
            "codesAlertBelow": codes_alert_below,
            "codesLeft": codes_left,
            "codesUploaded": codes_uploaded,
            "count": count,
            "couponOfferId": coupon_offer_id,
            "couponOfferName": coupon_offer_name,
            "couponValidDays": coupon_valid_days,
            "downloadUrl": download_url,
            "email": email,
            "health": health,
            "id": id,
            "rewardDescription": reward_description,
            "rewardName": reward_name,
            "stats": stats,
            "type": type_,
            "webhookSecretSet": webhook_secret_set,
            "webhookUrl": webhook_url,
        })
        if file is not UNSET:
            field_dict["file"] = file
        if image is not UNSET:
            field_dict["image"] = image
        if webhook_called_at is not UNSET:
            field_dict["webhookCalledAt"] = webhook_called_at
        if webhook_status is not UNSET:
            field_dict["webhookStatus"] = webhook_status

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.referral_file import ReferralFile # noqa: PLC0415
        from ..models.referral_image import ReferralImage # noqa: PLC0415
        from ..models.referral_milestone_email import ReferralMilestoneEmail # noqa: PLC0415
        from ..models.referral_milestone_health import ReferralMilestoneHealth # noqa: PLC0415
        from ..models.referral_milestone_stats import ReferralMilestoneStats # noqa: PLC0415
        d = dict(src_dict)
        ask_address = d.pop("askAddress")

        codes_alert_below = d.pop("codesAlertBelow")

        codes_left = d.pop("codesLeft")

        codes_uploaded = d.pop("codesUploaded")

        count = d.pop("count")

        coupon_offer_id = d.pop("couponOfferId")

        coupon_offer_name = d.pop("couponOfferName")

        coupon_valid_days = d.pop("couponValidDays")

        download_url = d.pop("downloadUrl")

        email = ReferralMilestoneEmail.from_dict(d.pop("email"))




        health = ReferralMilestoneHealth.from_dict(d.pop("health"))




        id = d.pop("id")

        reward_description = d.pop("rewardDescription")

        reward_name = d.pop("rewardName")

        stats = ReferralMilestoneStats.from_dict(d.pop("stats"))




        type_ = ReferralMilestoneViewType(d.pop("type"))




        webhook_secret_set = d.pop("webhookSecretSet")

        webhook_url = d.pop("webhookUrl")

        _file = d.pop("file", UNSET)
        file: ReferralFile | Unset
        if isinstance(_file,  Unset):
            file = UNSET
        else:
            file = ReferralFile.from_dict(_file)




        _image = d.pop("image", UNSET)
        image: ReferralImage | Unset
        if isinstance(_image,  Unset):
            image = UNSET
        else:
            image = ReferralImage.from_dict(_image)




        _webhook_called_at = d.pop("webhookCalledAt", UNSET)
        webhook_called_at: datetime.datetime | Unset
        if isinstance(_webhook_called_at,  Unset):
            webhook_called_at = UNSET
        else:
            webhook_called_at = datetime.datetime.fromisoformat(_webhook_called_at)




        webhook_status = d.pop("webhookStatus", UNSET)

        referral_milestone_view = cls(
            ask_address=ask_address,
            codes_alert_below=codes_alert_below,
            codes_left=codes_left,
            codes_uploaded=codes_uploaded,
            count=count,
            coupon_offer_id=coupon_offer_id,
            coupon_offer_name=coupon_offer_name,
            coupon_valid_days=coupon_valid_days,
            download_url=download_url,
            email=email,
            health=health,
            id=id,
            reward_description=reward_description,
            reward_name=reward_name,
            stats=stats,
            type_=type_,
            webhook_secret_set=webhook_secret_set,
            webhook_url=webhook_url,
            file=file,
            image=image,
            webhook_called_at=webhook_called_at,
            webhook_status=webhook_status,
        )

        return referral_milestone_view

