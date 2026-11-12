from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.referral_check_key import ReferralCheckKey
from ..models.referral_check_status import ReferralCheckStatus
from typing import cast

if TYPE_CHECKING:
  from ..models.referral_check_file import ReferralCheckFile





T = TypeVar("T", bound="ReferralCheck")



@_attrs_define
class ReferralCheck:
    """ 
        Attributes:
            account (str):
            count (int):
            counts (list[int]):
            emails (int):
            files (list[ReferralCheckFile]):
            key (ReferralCheckKey):
            missing (list[str]):
            offer (str):
            rewards (list[str]):
            sender (str):
            status (ReferralCheckStatus):
     """

    account: str
    count: int
    counts: list[int]
    emails: int
    files: list[ReferralCheckFile]
    key: ReferralCheckKey
    missing: list[str]
    offer: str
    rewards: list[str]
    sender: str
    status: ReferralCheckStatus





    def to_dict(self) -> dict[str, Any]:
        from ..models.referral_check_file import ReferralCheckFile # noqa: PLC0415
        account = self.account

        count = self.count

        counts = self.counts



        emails = self.emails

        files = []
        for files_item_data in self.files:
            files_item = files_item_data.to_dict()
            files.append(files_item)



        key = self.key.value

        missing = self.missing



        offer = self.offer

        rewards = self.rewards



        sender = self.sender

        status = self.status.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "account": account,
            "count": count,
            "counts": counts,
            "emails": emails,
            "files": files,
            "key": key,
            "missing": missing,
            "offer": offer,
            "rewards": rewards,
            "sender": sender,
            "status": status,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.referral_check_file import ReferralCheckFile # noqa: PLC0415
        d = dict(src_dict)
        account = d.pop("account")

        count = d.pop("count")

        counts = cast(list[int], d.pop("counts"))


        emails = d.pop("emails")

        files = []
        _files = d.pop("files")
        for files_item_data in (_files):
            files_item = ReferralCheckFile.from_dict(files_item_data)



            files.append(files_item)


        key = ReferralCheckKey(d.pop("key"))




        missing = cast(list[str], d.pop("missing"))


        offer = d.pop("offer")

        rewards = cast(list[str], d.pop("rewards"))


        sender = d.pop("sender")

        status = ReferralCheckStatus(d.pop("status"))




        referral_check = cls(
            account=account,
            count=count,
            counts=counts,
            emails=emails,
            files=files,
            key=key,
            missing=missing,
            offer=offer,
            rewards=rewards,
            sender=sender,
            status=status,
        )

        return referral_check

