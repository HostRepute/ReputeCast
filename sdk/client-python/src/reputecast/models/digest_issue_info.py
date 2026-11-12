from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.digest_issue_info_hold_reason import DigestIssueInfoHoldReason
from ..models.digest_issue_info_mode import DigestIssueInfoMode
from ..models.digest_issue_info_status import DigestIssueInfoStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.digest_feed_item import DigestFeedItem
  from ..models.digest_issue_since import DigestIssueSince
  from ..models.digest_person import DigestPerson





T = TypeVar("T", bound="DigestIssueInfo")



@_attrs_define
class DigestIssueInfo:
    """ 
        Attributes:
            campaign_id (str):
            digest_id (str):
            digest_name (str):
            issue_id (str):
            mode (DigestIssueInfoMode):
            number (int):
            posts (list[DigestFeedItem]):
            release_helpers (list[str]):
            status (DigestIssueInfoStatus):
            deadline (datetime.datetime | Unset):
            hold_reason (DigestIssueInfoHoldReason | Unset):
            owner (DigestPerson | Unset):
            prepared_at (datetime.datetime | Unset):
            previous_campaign_id (str | Unset):
            previous_number (int | Unset):
            send_at (datetime.datetime | Unset):
            since (DigestIssueSince | Unset):
     """

    campaign_id: str
    digest_id: str
    digest_name: str
    issue_id: str
    mode: DigestIssueInfoMode
    number: int
    posts: list[DigestFeedItem]
    release_helpers: list[str]
    status: DigestIssueInfoStatus
    deadline: datetime.datetime | Unset = UNSET
    hold_reason: DigestIssueInfoHoldReason | Unset = UNSET
    owner: DigestPerson | Unset = UNSET
    prepared_at: datetime.datetime | Unset = UNSET
    previous_campaign_id: str | Unset = UNSET
    previous_number: int | Unset = UNSET
    send_at: datetime.datetime | Unset = UNSET
    since: DigestIssueSince | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.digest_feed_item import DigestFeedItem # noqa: PLC0415
        from ..models.digest_issue_since import DigestIssueSince # noqa: PLC0415
        from ..models.digest_person import DigestPerson # noqa: PLC0415
        campaign_id = self.campaign_id

        digest_id = self.digest_id

        digest_name = self.digest_name

        issue_id = self.issue_id

        mode = self.mode.value

        number = self.number

        posts = []
        for posts_item_data in self.posts:
            posts_item = posts_item_data.to_dict()
            posts.append(posts_item)



        release_helpers = self.release_helpers



        status = self.status.value

        deadline: str | Unset = UNSET
        if not isinstance(self.deadline, Unset):
            deadline = self.deadline.isoformat()

        hold_reason: str | Unset = UNSET
        if not isinstance(self.hold_reason, Unset):
            hold_reason = self.hold_reason.value


        owner: dict[str, Any] | Unset = UNSET
        if not isinstance(self.owner, Unset):
            owner = self.owner.to_dict()

        prepared_at: str | Unset = UNSET
        if not isinstance(self.prepared_at, Unset):
            prepared_at = self.prepared_at.isoformat()

        previous_campaign_id = self.previous_campaign_id

        previous_number = self.previous_number

        send_at: str | Unset = UNSET
        if not isinstance(self.send_at, Unset):
            send_at = self.send_at.isoformat()

        since: dict[str, Any] | Unset = UNSET
        if not isinstance(self.since, Unset):
            since = self.since.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "campaignId": campaign_id,
            "digestId": digest_id,
            "digestName": digest_name,
            "issueId": issue_id,
            "mode": mode,
            "number": number,
            "posts": posts,
            "releaseHelpers": release_helpers,
            "status": status,
        })
        if deadline is not UNSET:
            field_dict["deadline"] = deadline
        if hold_reason is not UNSET:
            field_dict["holdReason"] = hold_reason
        if owner is not UNSET:
            field_dict["owner"] = owner
        if prepared_at is not UNSET:
            field_dict["preparedAt"] = prepared_at
        if previous_campaign_id is not UNSET:
            field_dict["previousCampaignId"] = previous_campaign_id
        if previous_number is not UNSET:
            field_dict["previousNumber"] = previous_number
        if send_at is not UNSET:
            field_dict["sendAt"] = send_at
        if since is not UNSET:
            field_dict["since"] = since

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.digest_feed_item import DigestFeedItem # noqa: PLC0415
        from ..models.digest_issue_since import DigestIssueSince # noqa: PLC0415
        from ..models.digest_person import DigestPerson # noqa: PLC0415
        d = dict(src_dict)
        campaign_id = d.pop("campaignId")

        digest_id = d.pop("digestId")

        digest_name = d.pop("digestName")

        issue_id = d.pop("issueId")

        mode = DigestIssueInfoMode(d.pop("mode"))




        number = d.pop("number")

        posts = []
        _posts = d.pop("posts")
        for posts_item_data in (_posts):
            posts_item = DigestFeedItem.from_dict(posts_item_data)



            posts.append(posts_item)


        release_helpers = cast(list[str], d.pop("releaseHelpers"))


        status = DigestIssueInfoStatus(d.pop("status"))




        _deadline = d.pop("deadline", UNSET)
        deadline: datetime.datetime | Unset
        if isinstance(_deadline,  Unset):
            deadline = UNSET
        else:
            deadline = datetime.datetime.fromisoformat(_deadline)




        _hold_reason = d.pop("holdReason", UNSET)
        hold_reason: DigestIssueInfoHoldReason | Unset
        if isinstance(_hold_reason,  Unset):
            hold_reason = UNSET
        else:
            hold_reason = DigestIssueInfoHoldReason(_hold_reason)




        _owner = d.pop("owner", UNSET)
        owner: DigestPerson | Unset
        if isinstance(_owner,  Unset):
            owner = UNSET
        else:
            owner = DigestPerson.from_dict(_owner)




        _prepared_at = d.pop("preparedAt", UNSET)
        prepared_at: datetime.datetime | Unset
        if isinstance(_prepared_at,  Unset):
            prepared_at = UNSET
        else:
            prepared_at = datetime.datetime.fromisoformat(_prepared_at)




        previous_campaign_id = d.pop("previousCampaignId", UNSET)

        previous_number = d.pop("previousNumber", UNSET)

        _send_at = d.pop("sendAt", UNSET)
        send_at: datetime.datetime | Unset
        if isinstance(_send_at,  Unset):
            send_at = UNSET
        else:
            send_at = datetime.datetime.fromisoformat(_send_at)




        _since = d.pop("since", UNSET)
        since: DigestIssueSince | Unset
        if isinstance(_since,  Unset):
            since = UNSET
        else:
            since = DigestIssueSince.from_dict(_since)




        digest_issue_info = cls(
            campaign_id=campaign_id,
            digest_id=digest_id,
            digest_name=digest_name,
            issue_id=issue_id,
            mode=mode,
            number=number,
            posts=posts,
            release_helpers=release_helpers,
            status=status,
            deadline=deadline,
            hold_reason=hold_reason,
            owner=owner,
            prepared_at=prepared_at,
            previous_campaign_id=previous_campaign_id,
            previous_number=previous_number,
            send_at=send_at,
            since=since,
        )

        return digest_issue_info

