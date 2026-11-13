from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.website_comment_list_comments import WebsiteCommentListComments
from ..models.website_comment_list_review import WebsiteCommentListReview
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.website_comment import WebsiteComment
  from ..models.website_comment_counts import WebsiteCommentCounts
  from ..models.website_comment_post import WebsiteCommentPost





T = TypeVar("T", bound="WebsiteCommentList")



@_attrs_define
class WebsiteCommentList:
    """ 
        Attributes:
            comments (WebsiteCommentListComments):
            counts (WebsiteCommentCounts):
            items (list[WebsiteComment]):
            posts (list[WebsiteCommentPost]):
            publication (str):
            publication_mark (str):
            review (WebsiteCommentListReview):
            next_cursor (str | Unset):
     """

    comments: WebsiteCommentListComments
    counts: WebsiteCommentCounts
    items: list[WebsiteComment]
    posts: list[WebsiteCommentPost]
    publication: str
    publication_mark: str
    review: WebsiteCommentListReview
    next_cursor: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.website_comment import WebsiteComment # noqa: PLC0415
        from ..models.website_comment_counts import WebsiteCommentCounts # noqa: PLC0415
        from ..models.website_comment_post import WebsiteCommentPost # noqa: PLC0415
        comments = self.comments.value

        counts = self.counts.to_dict()

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        posts = []
        for posts_item_data in self.posts:
            posts_item = posts_item_data.to_dict()
            posts.append(posts_item)



        publication = self.publication

        publication_mark = self.publication_mark

        review = self.review.value

        next_cursor = self.next_cursor


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "comments": comments,
            "counts": counts,
            "items": items,
            "posts": posts,
            "publication": publication,
            "publicationMark": publication_mark,
            "review": review,
        })
        if next_cursor is not UNSET:
            field_dict["nextCursor"] = next_cursor

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.website_comment import WebsiteComment # noqa: PLC0415
        from ..models.website_comment_counts import WebsiteCommentCounts # noqa: PLC0415
        from ..models.website_comment_post import WebsiteCommentPost # noqa: PLC0415
        d = dict(src_dict)
        comments = WebsiteCommentListComments(d.pop("comments"))




        counts = WebsiteCommentCounts.from_dict(d.pop("counts"))




        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = WebsiteComment.from_dict(items_item_data)



            items.append(items_item)


        posts = []
        _posts = d.pop("posts")
        for posts_item_data in (_posts):
            posts_item = WebsiteCommentPost.from_dict(posts_item_data)



            posts.append(posts_item)


        publication = d.pop("publication")

        publication_mark = d.pop("publicationMark")

        review = WebsiteCommentListReview(d.pop("review"))




        next_cursor = d.pop("nextCursor", UNSET)

        website_comment_list = cls(
            comments=comments,
            counts=counts,
            items=items,
            posts=posts,
            publication=publication,
            publication_mark=publication_mark,
            review=review,
            next_cursor=next_cursor,
        )

        return website_comment_list

